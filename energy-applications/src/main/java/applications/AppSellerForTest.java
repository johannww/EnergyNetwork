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
import java.sql.Timestamp;
import java.util.Base64;
import java.util.LinkedList;
import java.util.List;
import java.util.Random;
import java.util.concurrent.CyclicBarrier;
import java.util.concurrent.TimeoutException;
import java.util.function.Consumer;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonReader;

import org.apache.commons.cli.CommandLine;
import org.hyperledger.fabric.gateway.Contract;
import org.hyperledger.fabric.gateway.ContractEvent;
import org.hyperledger.fabric.gateway.ContractException;
import org.hyperledger.fabric.gateway.Gateway;
import org.hyperledger.fabric.gateway.Identities;
import org.hyperledger.fabric.gateway.Identity;
import org.hyperledger.fabric.gateway.Network;
import org.hyperledger.fabric.gateway.Transaction;
import org.hyperledger.fabric.gateway.X509Identity;

import applications.argparser.ArgParserSeller;
import applications.identity.ApplicationIdentityProvider;
import applications.testargparser.ArgParserSellerTest;

public class AppSellerForTest {

    private static CommandLine cmd;
    private static String paymentUrl;

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
        postJsonToUrl(paymentUrl + "/requestpayment", post);

    }

    private static void requestPaymentForEnergyTransactions(String sellerFullName, X509Identity identity,
            String energyTransactionJson, List<PublishedSellBid> publishedBids) throws Exception {

        JsonReader reader = Json.createReader(new StringReader(energyTransactionJson));
        JsonArray energyTransactions = reader.readArray();

        for (int bidIndex = 0; bidIndex < publishedBids.size(); bidIndex++) {
            PublishedSellBid publishedBid = publishedBids.get(bidIndex);
            JsonArray sellBidEnergyTransactions = energyTransactions.get(bidIndex).asJsonArray();
            for (int i = 0; i < sellBidEnergyTransactions.size(); i++) {
                JsonObject energyTransaction = sellBidEnergyTransactions.get(i).asJsonObject();
                String token = energyTransaction.getString("token");
                requestEnergyPaymentForToken(sellerFullName, identity, token);
                publishedBid.energyQuantitySettled += energyTransaction.getJsonNumber("energyquantity").doubleValue();
            }
            if (publishedBid.isFullySatisfied()) {
                publishedBids.remove(publishedBid);
                bidIndex--;
            }
        }
    }

    private static void registerAuctionEventListener(Contract contract, X509Identity x509Id,
            List<PublishedSellBid> publishedBids, String sellerFullName) {

        Consumer<ContractEvent> auctionPerfomedListener = new Consumer<ContractEvent>() {

            @Override
            public void accept(ContractEvent t) {

                if (t.getName().equals("auctionPerformed")) {
                    try {
                        // prove to utility company
                        String[] sellBidNumbers = new String[publishedBids.size() + 1];
                        sellBidNumbers[0] = sellerFullName;
                        int sellBidIndex = 1;
                        for (PublishedSellBid publishedBid : publishedBids)
                            sellBidNumbers[sellBidIndex++] = Integer.toString(publishedBid.bidNumber);

                        byte[] response = contract
                                .createTransaction("getEnergyTransactionsFromSellBidNumbersTestContext")
                                .evaluate(sellBidNumbers);
                        String energyTransactionsJson = new String(response, StandardCharsets.UTF_8);
                        // requestPaymentForEnergyTransactions(sellerFullName, x509Id,
                        // energyTransactionsJson,
                        // publishedBids);
                    } catch (Exception e) {
                        System.out.println(e.getMessage());
                    }
                }
            }
        };
        contract.addContractListener(auctionPerfomedListener, "auctionPerformed");
    }

    public static void main(String[] args) throws Exception {

        // enroll args
        /*
         * args = new String[] { "-e", "-u", "seller1-ufsc", "-pw", "seller1-ufsc",
         * "-host", "https://localhost:7000", "--cacert",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\seller1\\msp\\cacerts\\0-0-0-0-7000.pem",
         * "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\seller1\\msp",
         * "-msp", "UFSC", "--sell", "-kwh", "10", "-price", "4", "-type", "solar" }; //
         * wallet path args args = new String[] { "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\seller1\\msp",
         * "-msp", "UFSC", "-u", "seller1-ufsc", "--sell", "-kwh", "10", "-price", "4",
         * "-type", "solar" }; // file path credentials args args = new String[] {
         * "--certificate",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\seller1\\msp\\signcerts\\cert.pem",
         * "--privatekey",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\seller1\\msp\\keystore\\key.pem",
         * "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\seller1\\msp",
         * "-msp", "UFSC", "-u", "seller1-ufsc", "--sell", "-kwh", "10", "-price", "4",
         * "-type", "solar" };
         * 
         * 
         * args = new String[] { "-msp", "UFSC", "--basedir",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork", "--sellers", "2",
         * "--publishinterval", "2000", "--publishquantity", "1", "--paymentcompanyurl",
         * "http://localhost:81" };
         */

        ArgParserSellerTest testParser = new ArgParserSellerTest();

        cmd = testParser.parseArgs(args);
        String cliApplicationStr = System.getenv("APPLICATION_INSTANCE_ID");
        int cliApplicationId = cliApplicationStr != null ? Integer.parseInt(cliApplicationStr) : 0;
        int THREAD_NUM = Integer.parseInt(cmd.getOptionValue("sellers"));
        String msp = cmd.getOptionValue("msp");
        String baseDir = cmd.getOptionValue("basedir");
        int interval = Integer.parseInt(cmd.getOptionValue("publishinterval"));
        int thirtyPercentInterval = interval / 3;
        int maxPublish = Integer.parseInt(cmd.getOptionValue("publishquantity"));
        paymentUrl = cmd.getOptionValue("paymentcompanyurl");
        String dockerPrefix = cmd.hasOption("dockernetwork") ? "docker-" : "";
        String awsPrefix = cmd.hasOption("awsnetwork") ? "aws-" : "";

        // parsing seller params
        ArgParserSeller sellerParser = new ArgParserSeller();
        Gateway.Builder builder;
        Identity identity;
        try {
            String sellerNameIdentity = "seller1";
            Path certPath = Paths.get(baseDir, "hyperledger", msp.toLowerCase(), "seller1", "msp", "signcerts",
                    "cert.pem");
            Path pkPath = Paths.get(baseDir, "hyperledger", msp.toLowerCase(), "seller1", "msp", "keystore", "key.pem");
            args = new String[] { "--certificate", certPath.toString(), "--privatekey", pkPath.toString(), "-msp",
                    "UFSC", "-u", String.format("%s-ufsc", sellerNameIdentity), "--sell", "-kwh", "10", "-price", "4",
                    "-type", "solar" };
            cmd = sellerParser.parseArgs(args);

            // get seller identity
            identity = ApplicationIdentityProvider.getX509Identity(cmd);

            // Path to a common connection profile describing the network.
            String mspLower = cmd.getOptionValue("msp").toLowerCase();
            Path networkConfigFile = Paths.get("cfgs",
                    String.format("%s%s%s-connection-tls.json", awsPrefix, dockerPrefix, mspLower));

            // Configure the gateway connection used to access the network.
            builder = Gateway.createBuilder().identity(identity).networkConfig(networkConfigFile)
                    .discovery((dockerPrefix.length() > 0) || (awsPrefix.length() > 0));
        } catch (Exception e) {
            e.printStackTrace();
            throw new Error(String.format("Exiting with exception: " + e.getMessage()));
        }

        // Create a gateway connection for all threads
        try (Gateway gateway = builder.connect()) {
            // Obtain a smart contract deployed on the network.
            Network network = gateway.getNetwork("canal");
            Contract contract = network.getContract("energy");
            CyclicBarrier threadsBarrier = new CyclicBarrier(THREAD_NUM);
            Thread[] threads = new Thread[THREAD_NUM + 1];

            for (int i = 1; i <= THREAD_NUM; i++) {

                final int finalI = i;
                Random rand = new Random();
                int randomInterval = (interval - thirtyPercentInterval)
                        + rand.nextInt(2 * thirtyPercentInterval);
                threads[i] = new Thread() {

                    int threadNum = finalI;
                    // CommandLine cmd;

                    public void run() {

                        try {

                            List<PublishedSellBid> publishedBids = new LinkedList<PublishedSellBid>();
                            String sellerFullName = String.format("seller%d-%s",
                                    threadNum + (cliApplicationId - 1) * THREAD_NUM,
                                    cmd.getOptionValue("msp").toLowerCase());
                            registerAuctionEventListener(contract, (X509Identity) identity, publishedBids,
                                    sellerFullName);

                            Transaction transaction = null;
                            try {
                                transaction = contract.createTransaction("registerSellerTestContext");
                                transaction.submit(sellerFullName, "2", "2");
                            } catch (Exception e) {
                                //System.out.println(String
                                        //.format("Seller %d probably already registered: " + e.getMessage(), threadNum));
                            }

                            long totalExecutionTime = 0, startExecution = 0, transactionTimeWait = 0,
                                    startTransaction = 0, singleSignatureTime = 0, startSignature;
                            String generationBeginningTime, generationEndTime, randomGeneratedEnergy;
                            generationBeginningTime = Long.toString(System.currentTimeMillis() / 1000L);
                            // generationBeginningTime = "0";

                            // hold all testing threads here until all are ready to go
                            threadsBarrier.await();
                            // Submit SellBid
                            int publish = 0;
                            // adding a little randomness to start time to avoid 100% sync among threads
                            Thread.sleep(new Random().nextInt(500) + 10000);
                            startExecution = System.currentTimeMillis();
                            int invalidatedEnergyGenerations = 0, invalidatedSellbid = 0;
                            while (publish < maxPublish) {

                                try {
                                    // calling register sellbid transaction publishEnergyGenerationTestContext
                                    startTransaction = System.currentTimeMillis();
                                    transaction = contract.createTransaction("publishEnergyGenerationTestContext");
                                    generationEndTime = Long.toString(System.currentTimeMillis() / 1000L);
                                    randomGeneratedEnergy = Double.toString(new Random().nextDouble() * 20 + 10);
                                    transaction.submit(sellerFullName, generationBeginningTime, generationEndTime,
                                            "solar", randomGeneratedEnergy);
                                    generationBeginningTime = Long.toString(System.currentTimeMillis() / 1000L);
                                    transactionTimeWait += System.currentTimeMillis() - startTransaction;

                                } catch (Exception e) {
                                    //System.out.println("Failed energy generation submission: " + e.getMessage());
                                    invalidatedEnergyGenerations++;
                                }

                                Thread.sleep(rand.nextInt(thirtyPercentInterval));

                                try {

                                    // calling register sellbid transaction
                                    startTransaction = System.currentTimeMillis();
                                    transaction = contract.createTransaction("registerSellBidTestContext");
                                    transaction.submit(sellerFullName, cmd.getOptionValue("energyquantitykwh"),
                                            cmd.getOptionValue("priceperkwh"), cmd.getOptionValue("energytype"));
                                    transactionTimeWait += System.currentTimeMillis() - startTransaction;
                                    publishedBids.add(new PublishedSellBid(publish + 1,
                                            Double.parseDouble(cmd.getOptionValue("energyquantitykwh"))));
                                } catch (Exception e) {
                                    //System.out.println("Failed SellBid submission: " + e.getMessage());
                                    invalidatedSellbid++;
                                }

                                publish++;
                                Thread.sleep(randomInterval);
                            }

                            totalExecutionTime = System.currentTimeMillis() - startExecution;

                            System.out.println(getClass().getName() + " Thread " + Integer.toString(threadNum)
                                    + " took " + Long.toString(transactionTimeWait) + "ms to submit "
                                    + Integer.toString(maxPublish) + " transactions of "
                                    + Long.toString(totalExecutionTime)
                                    + "ms total execution time. \nA single signature takes: "
                                    + Long.toString(singleSignatureTime) + "ms ");

                        } catch (Exception e) {
                            e.printStackTrace();
                        }
                    }
                };
                threads[i].start();
            }

            for (int i = 1; i <= THREAD_NUM; i++)
                threads[i].join();
        }

        System.out.println("ENDED in timestamp: " + Long.toString(System.currentTimeMillis() / 1000L));
        System.exit(0);
    }
}
