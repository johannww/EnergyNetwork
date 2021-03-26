package applications;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.io.StringReader;
import java.net.InetSocketAddress;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.SecureRandom;
import java.util.Arrays;
import java.util.Base64;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import javax.json.Json;
import javax.json.JsonArray;
import javax.json.JsonObject;
import javax.json.JsonReader;

import com.google.protobuf.ByteString;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;

import org.apache.commons.cli.CommandLine;
import org.apache.milagro.amcl.FP256BN.BIG;
import org.apache.milagro.amcl.FP256BN.ECP;
import org.hyperledger.fabric.gateway.Contract;
import org.hyperledger.fabric.gateway.Gateway;
import org.hyperledger.fabric.gateway.Identity;
import org.hyperledger.fabric.gateway.Network;
import org.hyperledger.fabric.protos.common.Common.ChannelHeader;
import org.hyperledger.fabric.protos.common.Common.Envelope;
import org.hyperledger.fabric.protos.common.Common.Header;
import org.hyperledger.fabric.protos.common.Common.HeaderType;
import org.hyperledger.fabric.protos.common.Common.Payload;
import org.hyperledger.fabric.protos.common.Common.SignatureHeader;
import org.hyperledger.fabric.protos.common.MspPrincipal.OrganizationUnit;
import org.hyperledger.fabric.protos.idemix.Idemix.IssuerPublicKey;
import org.hyperledger.fabric.protos.idemix.Idemix.NymSignature;
import org.hyperledger.fabric.protos.ledger.rwset.Rwset.TxReadWriteSet;
import org.hyperledger.fabric.protos.ledger.rwset.kvrwset.KvRwset.KVRWSet;
import org.hyperledger.fabric.protos.ledger.rwset.kvrwset.KvRwset.KVWrite;
import org.hyperledger.fabric.protos.msp.Identities.SerializedIdemixIdentity;
import org.hyperledger.fabric.protos.msp.Identities.SerializedIdentity;
import org.hyperledger.fabric.protos.peer.ProposalPackage.ChaincodeAction;
import org.hyperledger.fabric.protos.peer.ProposalResponsePackage.ProposalResponsePayload;
import org.hyperledger.fabric.protos.peer.TransactionPackage;
import org.hyperledger.fabric.protos.peer.TransactionPackage.ChaincodeActionPayload;
import org.hyperledger.fabric.protos.peer.TransactionPackage.ChaincodeEndorsedAction;
import org.hyperledger.fabric.protos.peer.TransactionPackage.TransactionAction;
import org.hyperledger.fabric.protos.peer.TransactionPackage.TxValidationCode;
import org.hyperledger.fabric.sdk.Channel;
import org.hyperledger.fabric.sdk.TransactionInfo;
import org.hyperledger.fabric.sdk.idemix.IdemixIssuerPublicKey;
import org.hyperledger.fabric.sdk.idemix.IdemixPseudonymSignature;

import applications.argparser.ArgParserUtility;
import applications.identity.ApplicationIdentityProvider;


public class AppUtility {

    private static CommandLine cmd;
    private static Network network;
    private static Map<String, Double> tokenEnergyDiscounted;
    private static Map<String, Integer> clientLastNonces;
    private static String UTILITY_NAME = "UFSC";
    static SecureRandom secureRandom;

    private static class DiscountRequestHandler implements HttpHandler {
        @Override
        public void handle(HttpExchange t) throws IOException {
            InputStream in = t.getRequestBody();
            String request = new String(in.readAllBytes());

            JsonReader reader = Json.createReader(new StringReader(request));
            JsonObject energyDiscountRequest = reader.readObject();

            String clientName = energyDiscountRequest.getString("clientname");
            String registerBuyBidTxID = energyDiscountRequest.getString("registerbuybidtxid");
            String ipkB64 = energyDiscountRequest.getString("ipkb64");
            String buyerProofSignatureB64 = energyDiscountRequest.getString("sigb64");

            IssuerPublicKey ipk = IssuerPublicKey.parseFrom(Base64.getDecoder().decode(ipkB64));
            byte[] buyerProofSignature = Base64.getDecoder().decode(buyerProofSignatureB64);

            double kwhDiscounted;

            try {
                kwhDiscounted = energyDiscountRequest(network, clientName, registerBuyBidTxID, ipk, buyerProofSignature);
            } catch (Exception e) {
                kwhDiscounted = 0;
            }

            String response = "The discounted amount of energy was " + Double.toString(kwhDiscounted) + " KWH";
            t.sendResponseHeaders(200, response.length());
            OutputStream os = t.getResponseBody();
            os.write(response.getBytes());
            os.close();
        }
    }

    private static class NonceRequestHandler implements HttpHandler {
        @Override
        public void handle(HttpExchange t) throws IOException {
            InputStream in = t.getRequestBody();
            String request = new String(in.readAllBytes());

            JsonReader reader = Json.createReader(new StringReader(request));
            JsonObject energyDiscountRequest = reader.readObject();

            String clientName = energyDiscountRequest.getString("clientname");

            String response;


            try {
                Integer nonce = secureRandom.nextInt();
                clientLastNonces.put(clientName, nonce);
                response = Integer.toString(nonce);
            } catch (Exception e) {
                response = "Failed to generate nonce for client " + clientName;
            }


            t.sendResponseHeaders(200, response.length());
            OutputStream os = t.getResponseBody();
            os.write(response.getBytes());
            os.close();
        }
    }

    private static SerializedIdemixIdentity getTransactionSerializedIdemixIdentity(TransactionInfo transactionInfo)
            throws Exception {

        // Retrive Transaction SignatureHeader with the creator signature information
        Envelope envelope = transactionInfo.getEnvelope();
        ByteString envelopePayloadBytes = envelope.getPayload();
        Payload envelopePayload = Payload.parseFrom(envelopePayloadBytes);
        Header headerPayload = envelopePayload.getHeader();
        ByteString signatureHeaderBytes = headerPayload.getSignatureHeader();
        SignatureHeader sigHeader = SignatureHeader.parseFrom(signatureHeaderBytes);

        // get Transaction Creator, to check if the buyer requesting the discount is the
        // creator of the registered BuyBid.
        ByteString transactionCreatorIdentityBytes = sigHeader.getCreator();
        SerializedIdentity serializedIdentityTransaction = SerializedIdentity
                .parseFrom(transactionCreatorIdentityBytes);
        ByteString serializedIdentityIdBytes = serializedIdentityTransaction.getIdBytes();
        SerializedIdemixIdentity serializedTransactionIdemixIdentity = SerializedIdemixIdentity
                .parseFrom(serializedIdentityIdBytes);

        return serializedTransactionIdemixIdentity;
    }

    private static boolean verifyBuyBidSignatureMatch(TransactionInfo transactionInfo, IssuerPublicKey ipk, byte[] msg,
            byte[] sig) throws Exception {

        // Retrieve transaction cretor's SerializedIdemixIdentity
        SerializedIdemixIdentity serializedTransactionIdemixIdentity = getTransactionSerializedIdemixIdentity(
                transactionInfo);

        // Mount BuyBid transaction creator's pseudonym
        BIG nymX = BIG.fromBytes(serializedTransactionIdemixIdentity.getNymX().toByteArray());
        BIG nymY = BIG.fromBytes(serializedTransactionIdemixIdentity.getNymY().toByteArray());
        ECP nym = new ECP(nymX, nymY);

        // getting ipkHash from transaction
        OrganizationUnit ou = OrganizationUnit.parseFrom(serializedTransactionIdemixIdentity.getOu());
        byte[] ipkHash = ou.getCertifiersIdentifier().toByteArray();

        // starting Idemix Signature Verifier for Signature 'sig'
        IdemixPseudonymSignature idemixSigVerifier = new IdemixPseudonymSignature(NymSignature.parseFrom(sig));

        // check if received Ipk has the same hash of the transaction 'ipkHash'
        if (Arrays.equals(ipkHash, ipk.getHash().toByteArray())) {
            // verifiy Signature 'sig' in relation to Message 'msg' and IssuerPublicKey
            // 'ipk'
            return (idemixSigVerifier.verify(nym, new IdemixIssuerPublicKey(ipk), msg));
        }

        return false;
    }

    private static List<KVWrite> getTransactionKVWriteSet(TransactionInfo transactionInfo) throws Exception {
        // Retrive Transaction KVWriteSet with the creator signature information
        Envelope envelope = transactionInfo.getEnvelope();
        ByteString envelopePayloadBytes = envelope.getPayload();
        Payload envelopePayload = Payload.parseFrom(envelopePayloadBytes);
        Header headerPayload = envelopePayload.getHeader();
        ChannelHeader channelHeader = ChannelHeader.parseFrom(headerPayload.getChannelHeader());

        int channelHeaderType = channelHeader.getType();

        if (HeaderType.forNumber(channelHeaderType) == HeaderType.ENDORSER_TRANSACTION) {
            TransactionPackage.Transaction transactionData = TransactionPackage.Transaction
                    .parseFrom(envelopePayload.getData());
            TransactionAction transactionAction = transactionData.getActions(0);
            ChaincodeActionPayload chaincodeActionPayload = ChaincodeActionPayload
                    .parseFrom(transactionAction.getPayload());
            ChaincodeEndorsedAction endorsedAction = chaincodeActionPayload.getAction();
            ProposalResponsePayload proposalResponsePayload = ProposalResponsePayload
                    .parseFrom(endorsedAction.getProposalResponsePayload());
            ChaincodeAction chaincodeAction = ChaincodeAction.parseFrom(proposalResponsePayload.getExtension());
            TxReadWriteSet readWriteSet = TxReadWriteSet.parseFrom(chaincodeAction.getResults());
            KVRWSet keyValueSet = KVRWSet.parseFrom(readWriteSet.getNsRwset(1).getRwset());
            return keyValueSet.getWritesList();
        }
        return null;
    }

    private static boolean transactionStoredABuyBid(List<KVWrite> kVWriteList) throws Exception {

        if (kVWriteList != null && kVWriteList.size() == 1) {
            KVWrite kv = kVWriteList.get(0);
            if (kv.getKey().startsWith(new String(new byte[] { 0 }, "UTF-8") + "BuyBid")) {
                return true;
            }
        }
        return false;
    }

    private static double verifyBuyBidWasMatchedInAuction(Contract contract, List<KVWrite> kVWriteList)
            throws Exception {

        byte[] queryResponse = null;

        if (kVWriteList != null && kVWriteList.size() == 1) {
            KVWrite kv = kVWriteList.get(0);
            JsonReader reader = Json.createReader(new StringReader(kv.getValue().toStringUtf8()));
            JsonObject buyBid = reader.readObject();

            String paymentCompany = buyBid.getString("msppaymentcompany");
            String token = buyBid.getString("token");

            try {
                queryResponse = contract.evaluateTransaction("getEnergyTransactionsFromPaymentToken", paymentCompany,
                        token);
            } catch (Exception e) {
                return 0.0;
            }

            String responseStr = new String(queryResponse, "UTF-8");
            reader = Json.createReader(new StringReader(responseStr));
            JsonArray energyTransactions = reader.readArray();

            if (energyTransactions.size() > 0) {

                if (energyTransactions.get(0).asJsonObject().getString("utilityid").equals(UTILITY_NAME)) {

                    double boughtKWH = 0;
                    for (int i = 0; i < energyTransactions.size(); i++) {
                        JsonObject energyTransaction = energyTransactions.get(i).asJsonObject();
                        boughtKWH += energyTransaction.getJsonNumber("energyquantity").doubleValue();
                    }

                    if (!tokenEnergyDiscounted.containsKey(paymentCompany + token)) {
                        tokenEnergyDiscounted.put(paymentCompany + token, 0.0);
                    }

                    double alreadyDiscounted = tokenEnergyDiscounted.get(paymentCompany + token);

                    if (boughtKWH > alreadyDiscounted) {
                        tokenEnergyDiscounted.put(paymentCompany + token, boughtKWH - alreadyDiscounted);
                        return boughtKWH - alreadyDiscounted;
                    }
                }

            }

        }
        return 0.0;
    }

    private static double energyDiscountRequest(Network network, String clientName, String registerBuyBidTxID,
            IssuerPublicKey ipk,
            byte[] txIDSig) throws Exception {

        double kwhDiscounted = 0;

        Contract contract = network.getContract("energy");
        Channel channel = network.getChannel();
        // Retrive 'registerBuyBidTxID' information
        TransactionInfo transactionInfo = channel.queryTransactionByID(registerBuyBidTxID);

        TxValidationCode transactionIsValid = transactionInfo.getValidationCode();

        Integer clientLastNonce = clientLastNonces.get(clientName);

        if (transactionIsValid == TxValidationCode.VALID) {

            List<KVWrite> kVWriteList = getTransactionKVWriteSet(transactionInfo);
            // verify if transaction is ACTUALLY a 'registerBuyBid' transaction
            // verify if client requesting discount is the creator of the transaction
            if (transactionStoredABuyBid(kVWriteList)
                    && verifyBuyBidSignatureMatch(transactionInfo, ipk, (registerBuyBidTxID+clientLastNonce.toString()).getBytes(), txIDSig)) {
                // verify if there is a EnergyTransaction registered for bid
                kwhDiscounted = verifyBuyBidWasMatchedInAuction(contract, kVWriteList);
            }

        }

        return kwhDiscounted;

    }

    public static void main(String[] args) throws Exception {

        // enroll args
        /*args = new String[] { "-e", "-u", "admin1-ufsc", "-pw", "admin1-ufsc", "-host", "https://localhost:7000",
                "--cacert",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp\\cacerts\\0-0-0-0-7000.pem",
                "-w", "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp", "-msp",
                "UFSC", "-port", "80"};
        // wallet path args
        args = new String[] { "-w",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp", "-msp",
                "UFSC", "-u", "admin1-ufsc", "-port", "80" };
        // file path credentials args
        args = new String[] { "--certificate",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp\\signcerts\\cert.pem",
                "--privatekey",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp\\keystore\\key.pem",
                "-w", "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp", "-msp",
                "UFSC", "-u", "admin1-ufsc", "-port", "80"};*/

        // parsing utility params
        ArgParserUtility utilityParser = new ArgParserUtility();
        cmd = utilityParser.parseArgs(args);
        
        
        // get the utility identity
        Identity identity = ApplicationIdentityProvider.getX509Identity(cmd);
        
        // Path to a common connection profile describing the network.
        String msp = cmd.getOptionValue("msp").toLowerCase();
        String dockerPrefix = cmd.hasOption("dockernetwork") ? "docker-" : "";
        Path networkConfigFile = Paths.get("cfgs", String.format("%s%s-connection-tls.json", dockerPrefix, msp));
        
        // Configure the gateway connection used to access the network.
        Gateway.Builder builder = Gateway.createBuilder().identity(identity).networkConfig(networkConfigFile).discovery(dockerPrefix.length()>0);
        

        // publishing the buybid
        // Create a gateway connection
        try {
            Gateway gateway = builder.connect();
            secureRandom = new SecureRandom();
            tokenEnergyDiscounted = new HashMap<String, Double>();
            clientLastNonces = new HashMap<String, Integer>();

            // Obtain a smart contract deployed on the network.
            network = gateway.getNetwork("canal");
            // Contract contract = network.getContract("energy");

            // listen on HTTPS SERVER
            HttpServer server = HttpServer.create(new InetSocketAddress(Integer.parseInt(cmd.getOptionValue("port"))), 0);
            server.createContext("/noncerequest", new NonceRequestHandler());
            server.createContext("/discountrequest", new DiscountRequestHandler());
            server.setExecutor(null);
            server.start();
            
            //Thread.currentThread().join();


        } catch (Exception e) {
            e.printStackTrace();
        }

        // save SOMEHOW the idemix params for proving the buybid to the utility company

    }
}
