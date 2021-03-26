package applications;

import java.nio.charset.StandardCharsets;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.sql.Timestamp;
import java.util.HashMap;
import java.util.Map;
import java.util.Random;

import org.apache.commons.cli.CommandLine;
import org.apache.commons.math3.util.Pair;
import org.hyperledger.fabric.gateway.Contract;
import org.hyperledger.fabric.gateway.Gateway;
import org.hyperledger.fabric.gateway.Identity;
import org.hyperledger.fabric.gateway.Network;
import org.hyperledger.fabric.gateway.Transaction;

import applications.argparser.ArgParserSensor;
import applications.identity.ApplicationIdentityProvider;

public class AppSensor {

    private static CommandLine cmd;
    private static Random rand = new Random();

    public static class SmartData {
        public byte version;
        public long unit;
        public long timestamp;
        public double value;
        public byte error;
        public byte confidence;
        public int dev;

        public SmartData(byte version, long unit, long timestamp, double value, byte error, byte confidence, int dev) {
            this.version = version;
            this.unit = unit;
            this.timestamp = timestamp;
            this.value = value;
            this.error = error;
            this.confidence = confidence;
            this.dev = dev;
        }

    }

    private static Map<Long, Pair<Double, Double>> unitRange = new HashMap<Long, Pair<Double, Double>>() {
        {
            // temperature Kelvin MOD = 0
            put(2224179556L, new Pair<Double, Double>(0.0, 100.0)); // I32
            put(2761050468L, new Pair<Double, Double>(0.0, 100.0)); // I64
            put(3297921380L, new Pair<Double, Double>(0.0, 100.0)); // F32
            put(3834792292L, new Pair<Double, Double>(0.0, 100.0)); // D64

            // candela MOD = 0
            put(2224179493L, new Pair<Double, Double>(0.0, 100.0)); // I32
            put(2761050405L, new Pair<Double, Double>(0.0, 100.0)); // I64
            put(3297921317L, new Pair<Double, Double>(0.0, 100.0)); // F32
            put(3834792229L, new Pair<Double, Double>(0.0, 100.0)); // D64

            // meter MOD = 0
            put(2224441636L, new Pair<Double, Double>(0.0, 100.0)); // I32
            put(2761312548L, new Pair<Double, Double>(0.0, 100.0)); // I64
            put(3298183460L, new Pair<Double, Double>(0.0, 100.0)); // F32
            put(3835054372L, new Pair<Double, Double>(0.0, 100.0)); // D64

            // meters per second MOD = 0
            put(2224437540L, new Pair<Double, Double>(0.0, 100.0)); // I32
            put(2761308452L, new Pair<Double, Double>(0.0, 100.0)); // I64
            put(3298179364L, new Pair<Double, Double>(0.0, 100.0)); // F32
            put(3835050276L, new Pair<Double, Double>(0.0, 100.0)); // D64

            // cubic meters per second MOD = 0
            put(2224961828L, new Pair<Double, Double>(0.0, 100.0)); // I32
            put(2761832740L, new Pair<Double, Double>(0.0, 100.0)); // I64
            put(3298703652L, new Pair<Double, Double>(0.0, 100.0)); // F32
            put(3835574564L, new Pair<Double, Double>(0.0, 100.0)); // D64

        }
    };

    private static SmartData getRandomSmartData(Long unit) {
        // Pair<Double, Double> range = unitRange.get(unit);
        Pair<Double, Double> range = new Pair<Double, Double>(0.0, 100.0);
        double value = rand.nextDouble() * range.getSecond() + range.getFirst();
        Long timestamp = new Timestamp(System.currentTimeMillis()).getTime();
        return new SmartData((byte) 1, unit, timestamp, value, (byte) 0, (byte) 1, 0);
    }

    public static void main(String[] args) throws Exception {

        // enroll args
        args = new String[] { "-e", "-u", "sensor1-ufsc", "-pw", "sensor1-ufsc", "-host", "https://localhost:7000",
                "--cacert",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\sensor1\\msp\\cacerts\\0-0-0-0-7000.pem",
                "-w", "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\sensor1\\msp", "-msp",
                "UFSC", "-unit", "3834792292" };

        // wallet path args
        args = new String[] { "-w",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\sensor1\\msp", "-msp",
                "UFSC", "-u", "sensor1-ufsc", "-unit", "3835050277" };

        // file path credentials args
        args = new String[] { "--certificate",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\sensor1\\msp\\signcerts\\cert.pem",
                "--privatekey",
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\sensor1\\msp\\keystore\\key.pem",
                "-w", "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\sensor1\\msp", "-msp",
                "UFSC", "-u", "sensor1-ufsc", "-unit", "3835050277", "--publishinterval", "2", "--publishquantity",
                "10" };

        // parsing sensor params
        ArgParserSensor sensorParser = new ArgParserSensor();
        cmd = sensorParser.parseArgs(args);
        
        // get the sensor's identity
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

            // Publish SmartData
            Long unit = Long.parseLong(cmd.getOptionValue("unit"));
            int interval = Integer.parseInt(cmd.getOptionValue("publishinterval")) * 1000;

            int maxPublish = 1;//Integer.parseInt(cmd.getOptionValue("publishquantity"));

            try {
                int publish = 0;

                while (publish < maxPublish) {
                    SmartData smartData = getRandomSmartData(unit);

                    Transaction transaction = contract.createTransaction("publishSensorData");
                    byte[] transactionResult = transaction.submit(Byte.toString(smartData.version),
                            Long.toString(smartData.unit), Long.toString(smartData.timestamp),
                            Double.toString(smartData.value), Byte.toString(smartData.error),
                            Byte.toString(smartData.confidence), Integer.toString(smartData.dev));

                    //Thread.sleep(1000);
                    //transaction = contract.createTransaction("auction");
                    //transactionResult = transaction.submit();                    

                    System.out.println(new String(transactionResult, StandardCharsets.UTF_8));
                    publish++;
                    Thread.sleep(interval);
                }
            } catch (Exception e) {
                e.printStackTrace();
            }

            System.out.println("ENDED!");

            // Evaluate transactions that query state from the ledger.
            // byte[] queryResponse = contract.evaluateTransaction("query", "A");
            // System.out.println(new String(queryResponse, StandardCharsets.UTF_8));

        } catch (Exception e) {// (ContractException e) {
            e.printStackTrace();
        }

    }
}
