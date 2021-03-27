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
import java.util.Random;
import java.util.concurrent.CyclicBarrier;
import java.util.function.Consumer;

import javax.json.Json;
import javax.json.JsonObject;

import com.google.protobuf.ByteString;

import org.apache.commons.cli.CommandLine;
import org.apache.milagro.amcl.FP256BN.BIG;
import org.hyperledger.fabric.gateway.Contract;
import org.hyperledger.fabric.gateway.ContractEvent;
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
import applications.testargparser.ArgParserBuyerTest;

public class AppBuyerForTest {

    private static CommandLine cmd;
    private static String utilityUrl, paymentUrl;

    private static class PublishedBuyBid {
        public String paymentCompanyId;
        public String paymentToken;
        public String transactionID;
        public IssuerPublicKey ipk;
        public IdemixSigningIdentity signingId;
        public double energyQuantityKWH;
        public double energyQuantitySettled;

        public PublishedBuyBid(String paymentCompanyId, String paymentToken, String transactionID, IssuerPublicKey ipk,
                IdemixSigningIdentity signingId, double energyQuantityKWH) {
            this.paymentCompanyId = paymentCompanyId;
            this.paymentToken = paymentToken;
            this.transactionID = transactionID;
            this.ipk = ipk;
            this.signingId = signingId;
            this.energyQuantityKWH = energyQuantityKWH;
            this.energyQuantitySettled = 0.0;
        }

        public boolean isFullySatisfied() {
            return this.energyQuantityKWH >= this.energyQuantitySettled;
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
        postJsonToUrl(paymentUrl + "/putfunds", post);
    }

    private static String requestPaymentToken() throws Exception {
        String token = "";
        JsonObject post = Json.createObjectBuilder().add("clientname", "buyer1-idemixorg").add("funds", 1000).build();
        token = postJsonToUrl(paymentUrl + "/gettoken", post);
        return token;
    }

    private static void requestBuyBidValidation(String token) throws Exception {

        JsonObject post = Json.createObjectBuilder().add("clientname", "buyer1-idemixorg").add("token", token).build();
        postJsonToUrl(paymentUrl + "/validatebuybid", post);

    }

    private static void requestEnergyDiscount(String clientName, String registerBuyBidTxID, IssuerPublicKey ipk,
            byte[] buyerProvingPseudonymSignature) throws Exception {

        // String utilityHttpAddress = cmd.getOptionValue("utilityhttpaddress");
        String ipkB64 = Base64.getEncoder().encodeToString(ipk.toByteArray());
        String sigB64 = Base64.getEncoder().encodeToString(buyerProvingPseudonymSignature);

        JsonObject post = Json.createObjectBuilder().add("clientname", "buyer1-idemixorg")
                .add("registerbuybidtxid", registerBuyBidTxID).add("ipkb64", ipkB64).add("sigb64", sigB64).build();
        postJsonToUrl(utilityUrl + "/discountrequest", post);
    }

    private static int getUtilityCompanyNonce() throws Exception {
        JsonObject post = Json.createObjectBuilder().add("clientname", "buyer1-idemixorg").build();
        String response = postJsonToUrl(utilityUrl + "/noncerequest", post);
        return Integer.parseInt(response);
    }

    private static void registerAuctionEventListener(Contract contract, List<PublishedBuyBid> publishedBids,
            String buyerFullName) throws InvalidArgumentException {

        Consumer<ContractEvent> auctionPerfomedListener = new Consumer<ContractEvent>() {

            @Override
            public void accept(ContractEvent t) {

                if (t.getName().equals("auctionPerformed")) {
                    try {
                        // prove to utility company
                        for (PublishedBuyBid publishedBid : publishedBids) {
                            byte[] response = contract.createTransaction("transactionsEnergyQuantityFromPaymentToken")
                                    .evaluate(publishedBid.paymentCompanyId, publishedBid.paymentToken);

                            double energyQuantitySettledByTransactions = Double.parseDouble(new String(response));
                            if (energyQuantitySettledByTransactions > publishedBid.energyQuantitySettled) {
                                int utilityNonce = getUtilityCompanyNonce();
                                byte[] ipkOwnershipSignatureProof = publishedBid.signingId
                                        .sign((publishedBid.transactionID + Integer.toString(utilityNonce)).getBytes());
                                requestEnergyDiscount(buyerFullName, publishedBid.transactionID, publishedBid.ipk,
                                        ipkOwnershipSignatureProof);
                                publishedBid.energyQuantitySettled += energyQuantitySettledByTransactions;
                                if (publishedBid.isFullySatisfied())
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
        /*
         * args = new String[] { "-e", "-u", "buyer1-idemixorg", "-pw",
         * "buyer1-idemixorg", "-host", "https://localhost:7002", "-msp", "IDEMIXORG",
         * "-c",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\idemixorg\\buyer1\\msp\\cacerts\\0-0-0-0-7002.pem",
         * "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\idemixorg\\buyer1\\msp"
         * }; // wallet path args args = new String[] { "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\idemixorg\\buyer1\\msp",
         * "-msp", "IDEMIXORG", "-u", "buyer1-idemixorg", "-pci", "UFSC", "-token",
         * "tokentest1", "-kwh", "10", "-price", "50", "-type", "solar" }; // file path
         * credentials args args = new String[] { "-cp",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\idemixorg\\buyer1\\msp",
         * "-msp", "IDEMIXORG", "-u", "buyer1-idemixorg", "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\idemixorg\\buyer1\\msp",
         * "-pci", "UFSC", "-token", "tokentest1", "-kwh", "10", "-price", "50",
         * "-type", "solar" };
         * 
         * args = new String[] { "-msp", "IDEMIXORG", "--basedir",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork", "--buyers", "1",
         * "--publishinterval", "2", "--publishquantity", "50", "--utilityurl",
         * "http://localhost", "--paymentcompanyurl", "http://localhost:81" };
         */

        ArgParserBuyerTest testParser = new ArgParserBuyerTest();

        cmd = testParser.parseArgs(args);

        int THREAD_NUM = Integer.parseInt(cmd.getOptionValue("buyers"));
        String msp = cmd.getOptionValue("msp");
        String baseDir = cmd.getOptionValue("basedir");
        int interval = Integer.parseInt(cmd.getOptionValue("publishinterval"));
        int maxPublish = Integer.parseInt(cmd.getOptionValue("publishquantity"));
        paymentUrl = cmd.getOptionValue("paymentcompanyurl");
        utilityUrl = cmd.getOptionValue("utilityurl");
        String dockerPrefix = cmd.hasOption("dockernetwork") ? "docker-" : "";

        CyclicBarrier threadsBarrier = new CyclicBarrier(THREAD_NUM);
        Thread[] threads = new Thread[THREAD_NUM + 1];

        for (int i = 1; i <= THREAD_NUM; i++) {

            final int finalI = i;
            threads[i] = new Thread() {

                int threadNum = finalI;
                CommandLine cmd;

                public void run() {

                    // parsing buyer params
                    ArgParserBuyer buyerParser = new ArgParserBuyer();
                    Gateway.Builder builder;
                    IdemixIdentity idemixId;
                    try {
                        String buyerName = "buyer" + Integer.toString(threadNum);
                        Path idemixCredsPath = Paths.get(baseDir, "hyperledger", msp.toLowerCase(), buyerName, "msp");
                        String[] args = new String[] { "-cp", idemixCredsPath.toString(), "-msp", msp, "-u",
                                String.format("%s-%s", buyerName, msp.toLowerCase()), "-pci", "UFSC", "-kwh", "10",
                                "-price", "50", "-type", "solar", "-token", "tokentest1" };
                        cmd = buyerParser.parseArgs(args);

                        // get buyer's idemix identity
                        idemixId = ApplicationIdentityProvider.getIdemixIdentity(cmd);

                        // Path to a common connection profile describing the network.
                        String msp = cmd.getOptionValue("msp").toLowerCase();
                        Path networkConfigFile = Paths.get("cfgs",
                                String.format("%s%s-connection-tls.json", dockerPrefix, msp));

                        // Configure the gateway connection used to access the network.
                        builder = Gateway.createBuilder().identity(idemixId).networkConfig(networkConfigFile)
                                .discovery(dockerPrefix.length() > 0);
                    } catch (Exception e) {
                        e.printStackTrace();
                        throw new Error(
                                String.format("Thread %d exiting with exception: " + e.getMessage(), threadNum));
                    }
                    // publishing the buybid
                    // Create a gateway connection
                    try (Gateway gateway = builder.connect()) {

                        // Obtain a smart contract deployed on the network.
                        Network network = gateway.getNetwork("canal");
                        Contract contract = network.getContract("energy");

                        List<PublishedBuyBid> publishedBids = new LinkedList<PublishedBuyBid>();
                        String buyerFullName = cmd.getOptionValue("user");
                        registerAuctionEventListener(contract, publishedBids, buyerFullName);

                        long totalExecutionTime = 0, startExecution = 0, transactionTimeWait = 0, startTransaction = 0,
                                singleSignatureTime = 0, startSignature;

                        // Putting funds on buyer accounts to request token
                        putFundsOnPaymentAccount(1000000);

                        threadsBarrier.await();
                        startExecution = System.currentTimeMillis();

                        int publish = 0;
                        // adding a little randomness to start time to avoid 100% sync among threads
                        Thread.sleep(new Random().nextInt(500) + 2000);
                        while (publish < maxPublish) {
                            // Request token to Payment Company
                            String token = requestPaymentToken();

                            // Submit BuyBid
                            startTransaction = System.currentTimeMillis();
                            String paymentCompanyId = cmd.getOptionValue("paymentcompanyid");
                            String utilityCompanyId = "UFSC";
                            String energyQuantity = cmd.getOptionValue("energyquantitykwh");
                            String pricePerKwh = cmd.getOptionValue("priceperkwh");
                            String energyType = cmd.getOptionValue("energytype");
                            Transaction transaction = contract.createTransaction("registerBuyBid");
                            byte[] transactionResult = transaction.submit(paymentCompanyId, token, utilityCompanyId,
                                    energyQuantity, pricePerKwh, energyType);
                            transactionTimeWait += System.currentTimeMillis() - startTransaction;

                            // Request BuyBid validation to the Payment Company
                            requestBuyBidValidation(token);

                            TransactionContext transactionContext = transaction.getTransactionContext();

                            String transactionID = transaction.getTransactionId();

                            // get Ipk and signing identity to send to utility company for verification
                            IssuerPublicKey ipk = idemixId.getIpk().toProto();
                            IdemixSigningIdentity signingId = (IdemixSigningIdentity) transactionContext
                                    .getSigningIdentity();

                            // signingId.getNym();

                            publishedBids.add(new PublishedBuyBid(paymentCompanyId, token, transactionID, ipk,
                                    signingId, Double.parseDouble(energyQuantity)));

                            publish++;
                            Thread.sleep(interval);
                        }
                        totalExecutionTime = System.currentTimeMillis() - startExecution;

                        System.out.println(getClass().getName() + " Thread " + Integer.toString(threadNum) + " took "
                                + Long.toString(transactionTimeWait) + "ms to submit " + Integer.toString(maxPublish)
                                + " transactions of " + Long.toString(totalExecutionTime)
                                + "ms total execution time. \nA single signature takes: "
                                + Long.toString(singleSignatureTime) + "ms ");

                    } catch (Exception e) {
                        e.printStackTrace();
                        throw new Error(
                                String.format("Thread %d exiting with exception: " + e.getMessage(), threadNum));
                    }
                }
            };
            threads[i].start();
        }
        // save SOMEHOW the idemix params for proving the buybid to the utility company

        for (int i = 1; i <= THREAD_NUM; i++)
            threads[i].join();

        System.out.println("ENDED!");
        System.exit(0);
    }

}
