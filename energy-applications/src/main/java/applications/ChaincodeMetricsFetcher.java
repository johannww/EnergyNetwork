package applications;

import java.io.StringReader;
import java.io.StringWriter;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.HashMap;
import java.util.Map;

import org.apache.commons.cli.CommandLine;

import org.hyperledger.fabric.gateway.Contract;
import org.hyperledger.fabric.gateway.Gateway;
import org.hyperledger.fabric.gateway.Identity;
import org.hyperledger.fabric.gateway.Network;
import org.hyperledger.fabric.gateway.Transaction;
import applications.argparser.ArgParserPeriodicAuction;
import applications.identity.ApplicationIdentityProvider;
import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonReader;
import javax.json.JsonWriter;
import javax.json.JsonWriterFactory;
import javax.json.stream.JsonGenerator;

public class ChaincodeMetricsFetcher {

    private static CommandLine cmd;
    private static Network network;

    public static void main(String[] args) throws Exception {

        // enroll args
        /*
         * args = new String[] { "-e", "-u", "admin1-ufsc", "-pw", "admin1-ufsc",
         * "-host", "https://localhost:7000", "--cacert",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp\\cacerts\\0-0-0-0-7000.pem",
         * "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp",
         * "-msp", "UFSC", "-port", "80"}; // wallet path args args = new String[] {
         * "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp",
         * "-msp", "UFSC", "-u", "admin1-ufsc", "-port", "80" }; // file path
         * credentials args
         
        args = new String[] { "--certificate",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp\\signcerts\\cert.pem",
                "--privatekey",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp\\keystore\\key.pem",
                "-msp", "UFSC", "--auctioninterval", "100000", "--dockernetwork" };*/

        // parsing utility params
        ArgParserPeriodicAuction auctionParser = new ArgParserPeriodicAuction();
        cmd = auctionParser.parseArgs(args);

        // get the utility identity
        Identity identity = ApplicationIdentityProvider.getX509Identity(cmd);

        // Path to a common connection profile describing the network.
        String dockerPrefix = cmd.hasOption("dockernetwork") ? "docker-" : "";
        String awsPrefix = cmd.hasOption("awsnetwork") ? "aws-" : "";
        String mspLower = cmd.getOptionValue("msp").toLowerCase();
        Path networkConfigFile = Paths.get("cfgs",
                String.format("%s%s%s-connection-tls.json", awsPrefix, dockerPrefix, mspLower));

        // Configure the gateway connection used to access the network.
        Gateway.Builder builder = Gateway.createBuilder().identity(identity).networkConfig(networkConfigFile)
                .discovery((dockerPrefix.length() > 0) || (awsPrefix.length() > 0));

        // publishing the buybid
        // Create a gateway connection
        try (Gateway gateway = builder.connect()) {

            // Obtain a smart contract deployed on the network.
            network = gateway.getNetwork("canal");
            Contract contract = network.getContract("energy");

            Transaction transaction = contract.createTransaction("getAverageFunctionTimes");
            byte[] response = transaction.evaluate(args);
            JsonReader reader = Json.createReader(new StringReader(new String(response)));
            JsonObject chaincodeAverageFunctionsTimes = reader.readObject();

            StringWriter sw = new StringWriter();
            Map<String, Object> map = new HashMap<>();
            map.put(JsonGenerator.PRETTY_PRINTING, true);
            JsonWriterFactory writerFactory = Json.createWriterFactory(map);
            JsonWriter jsonWriter = writerFactory.createWriter(sw);
            jsonWriter.writeObject(chaincodeAverageFunctionsTimes);
            jsonWriter.close();
            
            System.out.println(sw.toString());

        } catch (Exception e) {
            e.printStackTrace();
        }

        // save SOMEHOW the idemix params for proving the buybid to the utility company

    }

}
