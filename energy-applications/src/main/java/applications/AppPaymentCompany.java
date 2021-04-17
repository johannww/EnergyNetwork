package applications;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.io.StringReader;
import java.net.InetSocketAddress;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.InvalidKeyException;
import java.security.NoSuchAlgorithmException;
import java.security.Signature;
import java.security.SignatureException;
import java.security.cert.CertificateEncodingException;
import java.security.cert.CertificateException;
import java.security.cert.X509Certificate;
import java.sql.Timestamp;
import java.util.ArrayList;
import java.util.Base64;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Random;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonReader;

import com.google.protobuf.ByteString;
import com.google.protobuf.InvalidProtocolBufferException;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;

import org.apache.commons.cli.CommandLine;
import org.apache.commons.math3.util.Pair;
import org.hyperledger.fabric.gateway.Contract;
import org.hyperledger.fabric.gateway.DefaultCommitHandlers;
import org.hyperledger.fabric.gateway.Gateway;
import org.hyperledger.fabric.gateway.Identities;
import org.hyperledger.fabric.gateway.Identity;
import org.hyperledger.fabric.gateway.Network;
import org.hyperledger.fabric.protos.common.Configtx.Config;
import org.hyperledger.fabric.protos.common.Configtx.ConfigGroup;
import org.hyperledger.fabric.protos.common.Configtx.ConfigValue;
import org.hyperledger.fabric.protos.msp.MspConfigPackage.FabricMSPConfig;
import org.hyperledger.fabric.protos.msp.MspConfigPackage.MSPConfig;
import org.hyperledger.fabric.sdk.Channel;

import applications.argparser.ArgParserPaymentCompany;
import applications.identity.ApplicationIdentityProvider;

public class AppPaymentCompany {
    private static CommandLine cmd;
    private static Network network;
    private static Map<String, UserFunds> tokenUser;
    private static Map<String, UserFunds> clientNameUser;
    private static Map<String, PaidBids> sellersPaidBids;

    private static String COMPANY_NAME = "UFSC";
    private static Map<String, ConfigGroup> channelOrgsMap;

    private static class UserFunds {
        public String clientName;
        public double funds;

        public UserFunds(String clientName, double funds) {
            this.clientName = clientName;
            this.funds = funds;
        }
    }

    private static class PaidBids extends ArrayList<Pair<String, Long>> {
        public PaidBids() {
            super();
        }
    }

    private static class PutFundsHandler implements HttpHandler {
        @Override
        public void handle(HttpExchange t) throws IOException {
            InputStream in = t.getRequestBody();
            String request = new String(in.readAllBytes());

            JsonReader reader = Json.createReader(new StringReader(request));
            JsonObject tokenRequest = reader.readObject();

            String clientName = tokenRequest.getString("clientname");
            double funds = tokenRequest.getJsonNumber("funds").doubleValue();

            String response;

            try {
                putFunds(clientName, funds);
                response = "Funds added to " + clientName;
                t.sendResponseHeaders(200, response.length());
            } catch (Exception e) {
                response = e.getMessage();
                t.sendResponseHeaders(404, response.length());
            }

            OutputStream os = t.getResponseBody();
            os.write(response.getBytes());
            os.close();
        }
    }

    private static class GetTokenHandler implements HttpHandler {
        @Override
        public void handle(HttpExchange t) throws IOException {
            InputStream in = t.getRequestBody();
            String request = new String(in.readAllBytes());

            JsonReader reader = Json.createReader(new StringReader(request));
            JsonObject tokenRequest = reader.readObject();

            String clientName = tokenRequest.getString("clientname");
            double funds = tokenRequest.getJsonNumber("funds").doubleValue();

            String response, token = "";

            try {
                token = getToken(clientName, funds);
                response = token;
                t.sendResponseHeaders(200, response.length());
            } catch (Exception e) {
                response = e.getMessage();
                t.sendResponseHeaders(404, response.length());
            }

            OutputStream os = t.getResponseBody();
            os.write(response.getBytes());
            os.close();
        }
    }

    private static class ValidateBuyBidHandler implements HttpHandler {
        @Override
        public void handle(HttpExchange t) throws IOException {
            InputStream in = t.getRequestBody();
            String request = new String(in.readAllBytes());

            JsonReader reader = Json.createReader(new StringReader(request));
            JsonObject tokenRequest = reader.readObject();

            String clientName = tokenRequest.getString("clientname");
            String token = tokenRequest.getString("token");

            String response;

            try {
                response = validateBuyBid(clientName, token);
                t.sendResponseHeaders(200, response.length());
            } catch (Exception e) {
                response = "Exception in buybid validation";
                t.sendResponseHeaders(404, response.length());
            }

            OutputStream os = t.getResponseBody();
            os.write(response.getBytes());
            os.close();
        }
    }

    private static class RequestPaymentHandler implements HttpHandler {
        @Override
        public void handle(HttpExchange t) throws IOException {
            InputStream in = t.getRequestBody();
            String request = new String(in.readAllBytes());

            JsonReader reader = Json.createReader(new StringReader(request));
            JsonObject paymentRequest = reader.readObject();

            String sellerName = paymentRequest.getString("sellername");
            String sellerMspID = paymentRequest.getString("mspseller");
            String token = paymentRequest.getString("token");

            String response = "";

            try {
                X509Certificate sellerCertificate = Identities
                        .readX509Certificate(paymentRequest.getString("certificate"));
                byte[] tokenSignature = Base64.getDecoder().decode(paymentRequest.getString("sigb64"));
                paySeller(sellerName, sellerMspID, token, sellerCertificate, tokenSignature);
                t.sendResponseHeaders(200, response.length());
            } catch (Exception e) {
                t.sendResponseHeaders(404, response.length());
            }

            OutputStream os = t.getResponseBody();
            os.write(response.getBytes());
            os.close();
        }
    }

    private static void putFunds(String clientName, double funds) {
        UserFunds userFunds = clientNameUser.get(clientName);
        if (userFunds != null) {
            userFunds.funds += funds;
        } else {
            clientNameUser.put(clientName, new UserFunds(clientName, funds));
        }
    }

    private static String getToken(String clientName, double funds) throws Exception {

        UserFunds userFunds = clientNameUser.get(clientName);
        if (userFunds != null && userFunds.funds < funds)
            throw new Exception("No funds");

        // generate Random String
        byte[] auxBytes = new byte[16];
        new Random().nextBytes(auxBytes);
        String token = new String(Base64.getEncoder().encode(auxBytes));

        // concatenate with the timestamp
        token += Long.toString(new Timestamp(System.currentTimeMillis()).getTime());

        // save in the relation token-user
        if (true)// (!tokenUser.containsKey(token))
            tokenUser.put(token, new UserFunds(clientName, funds));

        return token;
    }

    private static String validateBuyBid(String clientName, String token) throws Exception {

        // verify if token belongs to user
        UserFunds userFunds = tokenUser.get(token);

        if (clientName.equals(userFunds.clientName)) {
            Contract contract = network.getContract("energy");

            byte[] transactionResult = contract.createTransaction("validateBuyBid").submit(token,
                    Double.toString(userFunds.funds));

            return new String(transactionResult);

        }
        return "User " + clientName + " do not own the token " + token;

    }

    private static void setMspConfigValue() throws Exception {
        Channel channel = network.getChannel();
        Config config = Config.parseFrom(channel.getChannelConfigurationBytes());
        ConfigGroup channelGroup = config.getChannelGroup();
        channelOrgsMap = channelGroup.getGroupsMap().get("Application").getGroupsMap();

    }

    private static boolean verifySellerSignature(String sellerMspID, String token, X509Certificate sellerCertificate,
            byte[] tokenSignature)
            throws NoSuchAlgorithmException, CertificateEncodingException, InvalidKeyException, SignatureException {
        Signature signature = Signature.getInstance(sellerCertificate.getSigAlgName());
        signature.initVerify(sellerCertificate.getPublicKey());
        signature.update(token.getBytes());
        return signature.verify(tokenSignature);
    }

    private static boolean verifyRootCaSignature(String sellerMspID, X509Certificate sellerCertificate)
            throws InvalidProtocolBufferException, CertificateException {

        ConfigValue configValue = channelOrgsMap.get(sellerMspID).getValuesMap().get("MSP");
        MSPConfig mspConfig = MSPConfig.parseFrom(configValue.getValue());
        FabricMSPConfig fabricMSPConfig = FabricMSPConfig.parseFrom(mspConfig.getConfig());
        List<ByteString> rootCerts = fabricMSPConfig.getRootCertsList();

        for (ByteString rootCert : rootCerts) {
            X509Certificate x509RootCert = Identities.readX509Certificate(rootCert.toStringUtf8());
            try {
                sellerCertificate.verify(x509RootCert.getPublicKey());
                return true;
            } catch (Exception e) {
            }
        }
        return false;
    }

    private static String calculateSellerId(X509Certificate sellerCertificate) {
        String idStr = String.format("x509::%s::%s", sellerCertificate.getSubjectDN(), sellerCertificate.getIssuerDN());
        idStr = idStr.replace(", ", ",");
        idStr = idStr.replace(" + ", "+");
        return Base64.getEncoder().encodeToString(idStr.getBytes());
    }

    private static void paySeller(String sellerName, String sellerMspID, String token,
            X509Certificate sellerCertificate, byte[] tokenSignature) throws Exception {

        byte[] queryResponse = null;

        if (!verifySellerSignature(sellerMspID, token, sellerCertificate, tokenSignature))
            throw new Exception("Seller verify signature failed");

        if (!verifyRootCaSignature(sellerMspID, sellerCertificate))
            throw new Exception("Seller certificate was not signed by any Root CA");

        Contract contract = network.getContract("energy");

        // get EnergyTransactions with the claimed token
        queryResponse = contract.evaluateTransaction("getEnergyTransactionsFromPaymentToken", COMPANY_NAME, token);
        String responseStr = new String(queryResponse, "UTF-8");
        JsonReader reader = Json.createReader(new StringReader(responseStr));
        JsonArray energyTransactions = reader.readArray();

        if (energyTransactions.size() > 0) {
            double soldKWH, pricePerKWH, sellerPayment = 0;
            // String calculatedSellerId = calculateSellerId(sellerCertificate); REACTIVATE
            // THIS OUTSIDE TEST CONTEXT!!!!!
            String calculatedSellerId = sellerName;

            // get seller funds to increment with the payment
            UserFunds sellerFunds = clientNameUser.get(sellerName);
            if (sellerFunds == null) {
                sellerFunds = new UserFunds(sellerName, 0);
                clientNameUser.put(sellerName, sellerFunds);
            }

            for (int i = 0; i < energyTransactions.size(); i++) {
                JsonObject energyTransaction = energyTransactions.get(i).asJsonObject();
                String sellerMspTransaction = energyTransaction.getString("mspseller");
                String sellerIdTransaction = energyTransaction.getString("sellerid");

                // verify if EnergyTransaction has the same seller msp as the seller requesting
                // pay
                if (sellerMspID.equals(sellerMspTransaction) && sellerIdTransaction.equals(calculatedSellerId)) {
                    long sellBidNumber = Long.parseLong(energyTransaction.getString("sellerbidnumber"));
                    PaidBids paidBids = sellersPaidBids.get(sellerName);
                    if (paidBids == null) {
                        paidBids = new PaidBids();
                        sellersPaidBids.put(sellerName, paidBids);
                    }

                    Pair<String, Long> tokenSellBidNumber = new Pair<String, Long>(token, sellBidNumber);
                    if (!paidBids.contains(tokenSellBidNumber)) {
                        soldKWH = energyTransaction.getJsonNumber("energyquantity").doubleValue();
                        pricePerKWH = energyTransaction.getJsonNumber("priceperkwh").doubleValue();
                        sellerPayment += soldKWH * pricePerKWH;
                        paidBids.add(tokenSellBidNumber);
                        sellerFunds.funds += sellerPayment;
                    }
                }
            }
        }
    }

    public static void main(String[] args) throws Exception {

        // enroll args
        /*
         * args = new String[] { "-e", "-u", "admin1-ufsc", "-pw", "admin1-ufsc",
         * "-host", "https://localhost:7000", "--cacert",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp\\cacerts\\0-0-0-0-7000.pem",
         * "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp",
         * "-msp", "UFSC" }; // wallet path args args = new String[] { "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp",
         * "-msp", "UFSC", "-u", "admin1-ufsc" }; // file path credentials args
        args = new String[] { "--certificate",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp\\signcerts\\cert.pem",
                "--privatekey",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp\\keystore\\key.pem",
                "-w", "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp", "-msp",
                "UFSC", "-u", "admin1-ufsc", "-port", "81" };
         */
        // parsing payment company params
        ArgParserPaymentCompany pcParser = new ArgParserPaymentCompany();
        cmd = pcParser.parseArgs(args);

        // get the payment identity
        Identity identity = ApplicationIdentityProvider.getX509Identity(cmd);

        // Path to a common connection profile describing the network.
        String dockerPrefix = cmd.hasOption("dockernetwork") ? "docker-" : "";
        String awsPrefix = cmd.hasOption("awsnetwork") ? "aws-" : "";
        String mspLower = cmd.getOptionValue("msp").toLowerCase();
        Path networkConfigFile = Paths.get("cfgs",
                String.format("%s%s%s-connection-tls.json", awsPrefix, dockerPrefix, mspLower));

        // Configure the gateway connection used to access the network.
        Gateway.Builder builder = Gateway.createBuilder().identity(identity).networkConfig(networkConfigFile)
                .discovery((dockerPrefix.length() > 0) || (awsPrefix.length() > 0))
                .commitHandler(DefaultCommitHandlers.PREFER_MSPID_SCOPE_ANYFORTX);

        // publishing the buybid
        // Create a gateway connection
        try {
            Gateway gateway = builder.connect();
            tokenUser = new ConcurrentHashMap<String, UserFunds>();
            clientNameUser = new ConcurrentHashMap<String, UserFunds>();
            sellersPaidBids = new ConcurrentHashMap<String, PaidBids>();

            // Obtain a smart contract deployed on the network.
            network = gateway.getNetwork("canal");

            // loading map with root CAs certificate
            setMspConfigValue();

            /*
             * X509Certificate sellerCert =
             * Identities.readX509Certificate("-----BEGIN CERTIFICATE-----\n" +
             * "MIIC0DCCAnagAwIBAgIUZE0XYWPnGmx6CGCjoYNhcstjlCgwCgYIKoZIzj0EAwIw\n" +
             * "XjELMAkGA1UEBhMCVVMxFzAVBgNVBAgTDk5vcnRoIENhcm9saW5hMRQwEgYDVQQK\n" +
             * "EwtIeXBlcmxlZGdlcjEPMA0GA1UECxMGRmFicmljMQ8wDQYDVQQDEwZyY2EtY2Ew\n" +
             * "HhcNMjEwMjE5MTkyNzAwWhcNMjIwMjE5MTkzMjAwWjB2MQswCQYDVQQGEwJCUjEL\n" +
             * "MAkGA1UECBMCU0MxFjAUBgNVBAcTDUZsb3JpYW5vcG9saXMxDTALBgNVBAoTBFVG\n" +
             * "U0MxHDANBgNVBAsTBmNsaWVudDALBgNVBAsTBHVmc2MxFTATBgNVBAMTDHNlbGxl\n" +
             * "cjEtdWZzYzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABP9hAVnt0bi0z3rUmCaf\n" +
             * "gigukUYm+5+AnywJotRYgH/Yv88PFFNOpgfoen3UARliXhYCuFIDKgEpe6ZIxYad\n" +
             * "hOqjgfkwgfYwDgYDVR0PAQH/BAQDAgeAMAwGA1UdEwEB/wQCMAAwHQYDVR0OBBYE\n" +
             * "FPVziUe8D4LSKdGvP5yzL9gLMeQCMB8GA1UdIwQYMBaAFHV+JqFjw/wcgUJLBUPs\n" +
             * "uAtaPhUqMBoGA1UdEQQTMBGCD0RFU0tUT1AtQjdWMk8wQzB6BggqAwQFBgcIAQRu\n" +
             * "eyJhdHRycyI6eyJlbmVyZ3kuc2VsbGVyIjoidHJ1ZSIsImhmLkFmZmlsaWF0aW9u\n" +
             * "IjoidWZzYyIsImhmLkVucm9sbG1lbnRJRCI6InNlbGxlcjEtdWZzYyIsImhmLlR5\n" +
             * "cGUiOiJjbGllbnQifX0wCgYIKoZIzj0EAwIDSAAwRQIhAKMfAF3tSqzrHzronfEu\n" +
             * "QJRqB4N83t8R0DaKbKXdouVkAiAmY0y5qzi4g0u7KsA5EU18CGl2huQn4da+Xf8Y\n" +
             * "LOT9Jg==\n" + "-----END CERTIFICATE-----");
             * 
             * byte[] tokenSignature = Base64.getDecoder().decode(
             * "MEUCIQDyg/1n+zHAEOISKkkeBfmUPtwZLhCgCo40Y62/rt4NTQIgdLdfoNct3/KAw1xOO7EryV5Wdr/vm+6UI5BkpZwwicY="
             * );
             * 
             * paySeller("seller1", "UFSC", "tokentest1", sellerCert, tokenSignature);
             * paySeller("seller1", "UFSC", "tokentest1", sellerCert, tokenSignature);
             */
            // listen on HTTPS SERVER
            HttpServer server = HttpServer.create(new InetSocketAddress(Integer.parseInt(cmd.getOptionValue("port"))),
                    0);
            server.createContext("/putfunds", new PutFundsHandler());
            server.createContext("/gettoken", new GetTokenHandler());
            server.createContext("/validatebuybid", new ValidateBuyBidHandler());
            server.createContext("/requestpayment", new RequestPaymentHandler());
            int numberOfProcessors = Runtime.getRuntime().availableProcessors();
            // ExecutorService executor = Executors.newFixedThreadPool(numberOfProcessors);
            ExecutorService executor = Executors.newCachedThreadPool();
            server.setExecutor(executor);
            server.start();

            // Thread.currentThread().join();

        } catch (Exception e) {
            e.printStackTrace();
        }

    }
}
