package applications;

import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.net.URLConnection;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Base64;
import java.util.LinkedList;
import java.util.List;
import java.util.function.Consumer;

import javax.json.Json;
import javax.json.JsonObject;

import com.google.protobuf.ByteString;

import org.apache.commons.cli.CommandLine;
import org.apache.milagro.amcl.FP256BN.BIG;
import org.hyperledger.fabric.gateway.Contract;
import org.hyperledger.fabric.gateway.ContractEvent;
import org.hyperledger.fabric.gateway.ContractException;
import org.hyperledger.fabric.gateway.Gateway;
import org.hyperledger.fabric.gateway.IdemixIdentity;
import org.hyperledger.fabric.gateway.Network;
import org.hyperledger.fabric.gateway.Transaction;
import org.hyperledger.fabric.protos.idemix.Idemix.IssuerPublicKey;
import org.hyperledger.fabric.protos.msp.Identities.SerializedIdemixIdentity;
import org.hyperledger.fabric.protos.msp.Identities.SerializedIdentity;
import org.hyperledger.fabric.sdk.exception.InvalidArgumentException;
import org.hyperledger.fabric.sdk.identity.IdemixSigningIdentity;
import org.hyperledger.fabric.sdk.transaction.TransactionContext;

import applications.argparser.ArgParserBuyer;
import applications.identity.ApplicationIdentityProvider;

public class AppBuyer {

    private static CommandLine cmd;

    private static class PublishedBuyBid {
        public String paymentCompanyId;
        public String paymentToken;
        public String transactionID;
        public IssuerPublicKey ipk;
        public byte[] ipkOwnershipSignatureProof;

        public PublishedBuyBid(String paymentCompanyId, String paymentToken, String transactionID, IssuerPublicKey ipk,
                byte[] ipkOwnershipSignatureProof) {
            this.paymentCompanyId = paymentCompanyId;
            this.paymentToken = paymentToken;
            this.transactionID = transactionID;
            this.ipk = ipk;
            this.ipkOwnershipSignatureProof = ipkOwnershipSignatureProof;
        }
    }

    private static String postJsonToUrl(String urlStr, JsonObject post) throws Exception {
        String response = null;
        byte[] out = post.toString().getBytes();
        int length = out.length;

        URL url = new URL(urlStr);
        URLConnection con = url.openConnection();
        HttpURLConnection http = (HttpURLConnection) con;
        http.setRequestMethod("POST"); // PUT is another valid option
        http.setDoOutput(true);

        http.setFixedLengthStreamingMode(length);
        http.setRequestProperty("Content-Type", "application/json; charset=UTF-8");
        http.connect();
        try (OutputStream os = http.getOutputStream()) {
            os.write(out);
            os.close();
        }
        try (InputStream in = http.getInputStream()) {
            response = new String(in.readAllBytes());
            in.close();
        }
        return response;
    }

    private static void putFundsOnPaymentAccount(double funds) throws Exception {
        // String utilityHttpAddress = cmd.getOptionValue("utilityhttpaddress");

        JsonObject post = Json.createObjectBuilder().add("clientname", "buyer1-idemixorg").add("funds", funds).build();
        postJsonToUrl("http://localhost:81/putfunds", post);
    }

    private static String requestPaymentToken() throws Exception {

        String token = "";
        JsonObject post = Json.createObjectBuilder().add("clientname", "buyer1-idemixorg").add("funds", 1000).build();
        token = postJsonToUrl("http://localhost:81/gettoken", post);
        return token;
    }

    private static void requestBuyBidValidation(String token) throws Exception {

        JsonObject post = Json.createObjectBuilder().add("clientname", "buyer1-idemixorg").add("token", token).build();
        postJsonToUrl("http://localhost:81/validatebuybid", post);

    }

    private static void requestEnergyDiscount(String clientName, String registerBuyBidTxID, IssuerPublicKey ipk,
            byte[] buyerProvingPseudonymSignature) throws Exception {

        // String utilityHttpAddress = cmd.getOptionValue("utilityhttpaddress");
        String ipkB64 = Base64.getEncoder().encodeToString(ipk.toByteArray());
        String sigB64 = Base64.getEncoder().encodeToString(buyerProvingPseudonymSignature);

        JsonObject post = Json.createObjectBuilder().add("clientname", "buyer1-idemixorg")
                .add("registerbuybidtxid", registerBuyBidTxID).add("ipkb64", ipkB64).add("sigb64", sigB64).build();
        postJsonToUrl("http://localhost/discountrequest", post);
    }

    private static int getUtilityCompanyNonce() throws Exception {
        JsonObject post = Json.createObjectBuilder().add("clientname", "buyer1-idemixorg").build();
        String response = postJsonToUrl("http://localhost/noncerequest", post);
        return Integer.parseInt(response);
    }

    private static void registerAuctionEventListener(Contract contract, List<PublishedBuyBid> publishedBids)
            throws InvalidArgumentException {

        Consumer<ContractEvent> auctionPerfomedListener = new Consumer<ContractEvent>() {

            @Override
            public void accept(ContractEvent t) {

                if (t.getName().equals("auctionPerformed")) {
                    try {
                        // prove to utility company
                        for (PublishedBuyBid publishedBid : publishedBids) {
                            byte[] response = contract.createTransaction("energyTransactionsFromPaymentTokenExist")
                                    .evaluate(publishedBid.paymentCompanyId, publishedBid.paymentToken);

                            if (new String(response).equals("true")) {
                                requestEnergyDiscount("buyer1-idemixorg", publishedBid.transactionID, publishedBid.ipk,
                                        publishedBid.ipkOwnershipSignatureProof);
                                publishedBids.remove(publishedBid);
                            }
                        }
                    } catch (Exception e) {
                        System.out.println("HTTP FAILURE");
                    }
                }
            }
        };
        contract.addContractListener(auctionPerfomedListener, "auctionPerformed");
    }

    public static void main(String[] args) throws Exception {

        // enroll args
        args = new String[] { "-e", "-u", "buyer1-idemixorg", "-pw", "buyer1-idemixorg", "-host",
                "https://localhost:7002", "-msp", "IDEMIXORG", "-c",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\idemixorg\\buyer1\\msp\\cacerts\\0-0-0-0-7002.pem",
                "-w", "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\idemixorg\\buyer1\\msp" };
        // wallet path args
        args = new String[] { "-w",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\idemixorg\\buyer1\\msp", "-msp",
                "IDEMIXORG", "-u", "buyer1-idemixorg", "-pci", "UFSC", "-token", "tokentest1", "-kwh", "10", "-price",
                "50", "-type", "solar" };
        // file path credentials args
        args = new String[] { "-cp",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\idemixorg\\buyer1\\msp", "-msp",
                "IDEMIXORG", "-u", "buyer1-idemixorg", "-w",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\idemixorg\\buyer1\\msp", "-pci",
                "UFSC", "-token", "tokentest1", "-kwh", "10", "-price", "50", "-type", "solar" };

        // parsing buyer params
        ArgParserBuyer buyerParser = new ArgParserBuyer();
        cmd = buyerParser.parseArgs(args);

        // get buyer's idemix identity
        IdemixIdentity idemixId = ApplicationIdentityProvider.getIdemixIdentity(cmd);

        // Path to a common connection profile describing the network.
        String msp = cmd.getOptionValue("msp").toLowerCase();
        String dockerPrefix = cmd.hasOption("dockernetwork") ? "docker-" : "";
        Path networkConfigFile = Paths.get("cfgs", String.format("%s%s-connection-tls.json", dockerPrefix, msp));

        // Configure the gateway connection used to access the network.
        Gateway.Builder builder = Gateway.createBuilder().identity(idemixId).networkConfig(networkConfigFile).discovery(true);

        // publishing the buybid
        // Create a gateway connection
        try (Gateway gateway = builder.connect()) {

            // Obtain a smart contract deployed on the network.
            Network network = gateway.getNetwork("canal");
            Contract contract = network.getContract("energy");

            // enabling auction event listening
            List<PublishedBuyBid> publishedBids = new LinkedList<PublishedBuyBid>();
            registerAuctionEventListener(contract, publishedBids);

            // Putting funds on buyer accounts to request token
            //putFundsOnPaymentAccount(1000);

            // Request token to Payment Company
            //String token = requestPaymentToken();
            String token = "OI";

            // Submit BuyBid
            String paymentCompanyId = cmd.getOptionValue("paymentcompanyid");
            String utilityCompanyId = "UFSC";
            String energyAmount = cmd.getOptionValue("energyamountkwh");
            String pricePerKwh = cmd.getOptionValue("priceperkwh");
            String energyType = cmd.getOptionValue("energytype");
            Transaction transaction = contract.createTransaction("registerBuyBid");
            byte[] transactionResult = transaction.submit(paymentCompanyId, token, utilityCompanyId, energyAmount,
                    pricePerKwh, energyType);

            // Request BuyBid validation to the Payment Company
            requestBuyBidValidation(token);

            TransactionContext transactionContext = transaction.getTransactionContext();

            int utilityNonce = getUtilityCompanyNonce();

            String transactionID = transaction.getTransactionId();
            byte[] ipkOwnershipSignatureProof = transactionContext
                    .sign((transactionID + Integer.toString(utilityNonce)).getBytes());

            // GET TRANSACTION PSEUDONYM FROM CONTEXT TO MAYBE SAVE IT
            SerializedIdentity serializedIdentity = transactionContext.getIdentity();
            ByteString serializedIdBytes = serializedIdentity.getIdBytes();
            serializedIdentity.getMspidBytes();

            SerializedIdemixIdentity serializedIdemixIdentity = SerializedIdemixIdentity.parseFrom(serializedIdBytes);
            BIG nymXbuyer = BIG.fromBytes(serializedIdemixIdentity.getNymX().toByteArray());
            BIG nymYbuyer = BIG.fromBytes(serializedIdemixIdentity.getNymY().toByteArray());

            // get Ipk to send to utility company for verification
            IssuerPublicKey ipk = idemixId.getIpk().toProto();
            // IdemixSigningIdentity signingId = (IdemixSigningIdentity)
            // transactionContext.getSigningIdentity();

            // signingId.getNym();

            publishedBids
                    .add(new PublishedBuyBid(paymentCompanyId, token, transactionID, ipk, ipkOwnershipSignatureProof));

            System.out.println("TxID: " + transaction.getTransactionId());
            System.out.println("Ipk (Base64): " + Base64.getEncoder().encodeToString(ipk.toByteArray()));
            System.out.println("TxID Signature: " + Base64.getEncoder().encodeToString(ipkOwnershipSignatureProof));

            Thread.currentThread().join();

        } catch (ContractException e) {
            e.printStackTrace();
        }

        // save SOMEHOW the idemix params for proving the buybid to the utility company

    }

}
