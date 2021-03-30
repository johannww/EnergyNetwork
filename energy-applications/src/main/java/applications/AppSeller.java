package applications;

import java.io.InputStream;
import java.io.OutputStream;
import java.io.StringReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.net.URLConnection;
import java.nio.charset.StandardCharsets;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.Signature;
import java.security.cert.X509Certificate;
import java.util.Base64;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonReader;

import org.apache.commons.cli.CommandLine;
import org.hyperledger.fabric.gateway.Contract;
import org.hyperledger.fabric.gateway.ContractException;
import org.hyperledger.fabric.gateway.Gateway;
import org.hyperledger.fabric.gateway.Identities;
import org.hyperledger.fabric.gateway.Identity;
import org.hyperledger.fabric.gateway.Network;
import org.hyperledger.fabric.gateway.Transaction;
import org.hyperledger.fabric.gateway.X509Identity;

import applications.argparser.ArgParserSeller;
import applications.identity.ApplicationIdentityProvider;

public class AppSeller {

    private static CommandLine cmd;

    private static class PublishedSellBid {
        public int bidNumber;
        public double energyQuantityKWH;
        public double energyQuantitySettled;

        public PublishedSellBid(int bidNumber, double energyQuantityKWH) {
            this.bidNumber = bidNumber;
            this.energyQuantityKWH = energyQuantityKWH;
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

    private static void requestEnergyPaymentForToken(String sellerName, X509Identity identity, String token)
            throws Exception {

        // String utilityHttpAddress = cmd.getOptionValue("utilityhttpaddress");
        X509Certificate sellerCertificate = identity.getCertificate();
        Signature signature = Signature.getInstance(sellerCertificate.getSigAlgName());
        signature.initSign(identity.getPrivateKey());
        signature.update(token.getBytes());
        byte[] tokenSignature = signature.sign();
        String sigB64 = Base64.getEncoder().encodeToString(tokenSignature);

        JsonObject post = Json.createObjectBuilder().add("sellername", sellerName).add("mspseller", identity.getMspId())
                .add("token", token).add("certificate", Identities.toPemString(sellerCertificate)).add("sigb64", sigB64)
                .build();
        postJsonToUrl("http://localhost:81/requestpayment", post);

    }

    private static void requestPaymentForEnergyTransactions(String sellerName, X509Identity identity,
            String energyTransactionJson) throws Exception {

        JsonReader reader = Json.createReader(new StringReader(energyTransactionJson));
        JsonArray energyTransactions = reader.readArray();

        for (int i = 0; i < energyTransactions.size(); i++) {
            JsonObject energyTransaction = energyTransactions.get(i).asJsonObject();
            String token = energyTransaction.getString("token");
            requestEnergyPaymentForToken("seller1-ufsc", identity, token);
        }
    }

    /*private static void registerAuctionEventListener(Contract contract, X509Identity x509Id,
            List<PublishedSellBid> publishedBids, String sellerFullName) {

        Consumer<ContractEvent> auctionPerfomedListener = new Consumer<ContractEvent>() {

            @Override
            public void accept(ContractEvent t) {

                if (t.getName().equals("auctionPerformed")) {
                    try {
                        // prove to utility company
                        String[] sellBidNumbers = new String[publishedBids.size()];
                        int sellBidIndex = 0;
                        for (PublishedSellBid publishedBid : publishedBids)
                            sellBidNumbers[sellBidIndex++] = Integer.toString(publishedBid.bidNumber);

                        byte[] response = contract.createTransaction("getEnergyTransactionsFromSellBidNumbers")
                                .evaluate(sellBidNumbers);
                        String energyTransactionsJson = new String(response, StandardCharsets.UTF_8);
                        requestPaymentForEnergyTransactions(sellerFullName, x509Id, energyTransactionsJson,
                                publishedBids);
                    } catch (Exception e) {
                        System.out.println(e.getMessage());
                    }
                }
            }
        };
        contract.addContractListener(auctionPerfomedListener, "auctionPerformed");
    }*/

    public static void main(String[] args) throws Exception {

        // enroll args
        args = new String[] { "-e", "-u", "seller1-ufsc", "-pw", "seller1-ufsc", "-host", "https://localhost:7000",
                "--cacert",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\seller1\\msp\\cacerts\\0-0-0-0-7000.pem",
                "-w", "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\seller1\\msp", "-msp",
                "UFSC", "--sell", "-kwh", "10", "-price", "4", "-type", "solar" };
        // wallet path args
        args = new String[] { "-w",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\seller1\\msp", "-msp",
                "UFSC", "-u", "seller1-ufsc", "--sell", "-kwh", "10", "-price", "4", "-type", "solar" };
        // file path credentials args
        args = new String[] { "--certificate",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\seller1\\msp\\signcerts\\cert.pem",
                "--privatekey",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\seller1\\msp\\keystore\\key.pem",
                "-w", "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\seller1\\msp", "-msp",
                "UFSC", "-u", "seller1-ufsc", "--sell", "-kwh", "10", "-price", "4", "-type", "solar" };

        // parsing seller params
        ArgParserSeller sellerParser = new ArgParserSeller();
        cmd = sellerParser.parseArgs(args);

        // get seller identity
        Identity identity = ApplicationIdentityProvider.getX509Identity(cmd);

        // Path to a common connection profile describing the network.
        String msp = cmd.getOptionValue("msp").toLowerCase();
        String dockerPrefix = cmd.hasOption("dockernetwork") ? "docker-" : "";
        Path networkConfigFile = Paths.get("cfgs", String.format("%s%s-connection-tls.json", dockerPrefix, msp));

        // Configure the gateway connection used to access the network.
        Gateway.Builder builder = Gateway.createBuilder().identity(identity).networkConfig(networkConfigFile);

        // publishing the buybid
        // Create a gateway connection
        try (Gateway gateway = builder.connect()) {

            // Obtain a smart contract deployed on the network.
            Network network = gateway.getNetwork("canal");
            Contract contract = network.getContract("energy");

            // Submit SellBid

            Transaction transaction = contract.createTransaction("registerSellBid");
            byte[] transactionResult = transaction.submit(cmd.getOptionValue("energyamountkwh"),
                    cmd.getOptionValue("priceperkwh"), cmd.getOptionValue("energytype"));

            transaction = contract.createTransaction("getEnergyTransactionsFromFullSellBidKey");
            transactionResult = transaction.evaluate("UFSC",
                    "eDUwOTo6Q049c2VsbGVyMS11ZnNjLE9VPWNsaWVudCtPVT11ZnNjLE89VUZTQyxMPUZsb3JpYW5vcG9saXMsU1Q9U0MsQz1CUjo6Q049cmNhLWNhLE9VPUZhYnJpYyxPPUh5cGVybGVkZ2VyLFNUPU5vcnRoIENhcm9saW5hLEM9VVM=",
                    "995");
            String energyTransactionsJson = new String(transactionResult, StandardCharsets.UTF_8);
            System.out.println(energyTransactionsJson);
            X509Identity x509Id = (X509Identity) identity;

            requestPaymentForEnergyTransactions("seller1-ufsc", x509Id, energyTransactionsJson);

            // Evaluate transactions that query state from the ledger.
            // byte[] queryResponse = contract.evaluateTransaction("query", "A");
            // System.out.println(new String(queryResponse, StandardCharsets.UTF_8));

        } catch (ContractException e) {
            e.printStackTrace();
        }

    }
}
