package applications;

import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.HashMap;
import java.util.Map;
import java.util.Random;
import java.util.concurrent.CyclicBarrier;

import org.apache.commons.cli.CommandLine;
import org.apache.commons.math3.util.Pair;
import org.hyperledger.fabric.gateway.Contract;
import org.hyperledger.fabric.gateway.Gateway;
import org.hyperledger.fabric.gateway.Identity;
import org.hyperledger.fabric.gateway.Network;
import org.hyperledger.fabric.gateway.Transaction;
import org.hyperledger.fabric.sdk.exception.CryptoException;
import org.hyperledger.fabric.sdk.exception.InvalidArgumentException;
import org.hyperledger.fabric.sdk.transaction.TransactionContext;

import applications.argparser.ArgParserSensor;
import applications.identity.ApplicationIdentityProvider;
import applications.testargparser.ArgParserSensorTest;

public class AppSensorForTest {

    private static CommandLine cmd;

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

    private static SmartData getRandomSmartData(Long unit, int threadNum, int publishIndex) {
        // Pair<Double, Double> range = unitRange.get(unit);
        Pair<Double, Double> range = new Pair<Double, Double>(0.0, 100.0);
        double value = new Random().nextDouble() * range.getSecond() + range.getFirst();
        Long timestamp = System.currentTimeMillis() / 1000L;
        // Long timestamp = Long.parseLong(Integer.toString(threadNum) + "1111111" +
        // Integer.toString(publishIndex));
        return new SmartData((byte) 1, unit, timestamp, value, (byte) 0, (byte) 1, 0);
    }

    public static void main(String[] args) throws Exception {

        // enroll args
        /*
         * args = new String[] { "-e", "-u", "sensor1-ufsc", "-pw", "sensor1-ufsc",
         * "-host", "https://localhost:7000", "--cacert",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\sensor1\\msp\\cacerts\\0-0-0-0-7000.pem",
         * "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\sensor1\\msp",
         * "-msp", "UFSC", "-unit", "3834792292" };
         * 
         * // wallet path args args = new String[] { "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\sensor1\\msp",
         * "-msp", "UFSC", "-u", "sensor1-ufsc", "-unit", "3834792229" };
         * 
         * // file path credentials args args = new String[] { "--certificate",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\sensor1\\msp\\signcerts\\cert.pem",
         * "--privatekey",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\sensor1\\msp\\keystore\\key.pem",
         * "-w",
         * "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\sensor1\\msp",
         * "-msp", "UFSC", "-u", "sensor1-ufsc", "-unit", "3834792229",
         * "--publishinterval", "2", "--publishquantity", "50" };
         * 
         *  
          args = new String[] { "-msp", "UFSC", "--basedir",
          "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork", "--sensors", "100",
         "--unit", "3834792229", "--publishinterval", "2", "--publishquantity", "1" };*/
        

        ArgParserSensorTest testParser = new ArgParserSensorTest();

        cmd = testParser.parseArgs(args);

        int THREAD_NUM = Integer.parseInt(cmd.getOptionValue("sensors"));
        String msp = cmd.getOptionValue("msp");
        String baseDir = cmd.getOptionValue("basedir");
        Long unit = Long.parseLong(cmd.getOptionValue("unit"));
        int interval = Integer.parseInt(cmd.getOptionValue("publishinterval"));
        int maxPublish = Integer.parseInt(cmd.getOptionValue("publishquantity"));
        String dockerPrefix = cmd.hasOption("dockernetwork") ? "docker-" : "";
        String awsPrefix = cmd.hasOption("awsnetwork") ? "aws-" : "";

        // parsing sensor params
        ArgParserSensor sensorParser = new ArgParserSensor();
        Gateway.Builder builder;
        try {
            // file path credentials args
            String sensorNameIdentity = "sensor1";
            Path certPath = Paths.get(baseDir, "hyperledger", msp.toLowerCase(), "sensor1", "msp", "signcerts",
                    "cert.pem");
            Path pkPath = Paths.get(baseDir, "hyperledger", msp.toLowerCase(), "sensor1", "msp", "keystore", "key.pem");
            args = new String[] { "--certificate", certPath.toString(), "--privatekey", pkPath.toString(),
                    "-msp", "UFSC", "-u",  String.format(sensorNameIdentity+"-%s", cmd.getOptionValue("msp").toLowerCase()) };
            cmd = sensorParser.parseArgs(args);

            // get the sensor's identity
            Identity identity = ApplicationIdentityProvider.getX509Identity(cmd);

            // Path to a common connection profile describing the network.
            String mspLower = cmd.getOptionValue("msp").toLowerCase();
            Path networkConfigFile = Paths.get("cfgs", String.format("%s%s%s-connection-tls.json", awsPrefix, dockerPrefix, mspLower));

            // Configure the gateway connection used to access the network.
            builder = Gateway.createBuilder().identity(identity).networkConfig(networkConfigFile)
                    .discovery((dockerPrefix.length() > 0) || (awsPrefix.length() > 0));
        } catch (Exception e) {
            e.printStackTrace();
            throw new Error(String.format("Exiting with exception: " + e.getMessage()));
        }
        try (Gateway gateway = builder.connect()) {

            CyclicBarrier threadsBarrier = new CyclicBarrier(THREAD_NUM);
            Thread[] threads = new Thread[THREAD_NUM + 1];

            for (int i = 1; i <= THREAD_NUM; i++) {

                final int finalI = i;
                threads[i] = new Thread() {

                    int threadNum = finalI;
                    //CommandLine cmd;

                    public void run() {

                        // Create a gateway connection
                        try {

                            // Obtain a smart contract deployed on the network.
                            Network network = gateway.getNetwork("canal");
                            Contract contract = network.getContract("energy");

                            String sensorFullName = String.format("sensor%d-%s", threadNum, cmd.getOptionValue("msp").toLowerCase());

                            long totalExecutionTime = 0, startExecution = 0, transactionTimeWait = 0,
                                    startTransaction = 0, singleSignatureTime = 0, startSignature;
                            Transaction transaction = null;


                            try {
                                transaction = contract.createTransaction("sensorDeclareActiveTestContext");
                                transaction.submit(sensorFullName);
                            } catch (Exception e) {
                                System.out.println(String.format("Sensor %d probably already active: " + e.getMessage(),
                                        threadNum));
                            }

                            // hold all testing threads here until all are ready to go
                            threadsBarrier.await();
                            startExecution = System.currentTimeMillis();

                            // Publish SmartData
                            try {
                                int publish = 0;
                                // adding a little randomness to start time to avoid 100% sync among threads
                                Thread.sleep(new Random().nextInt(500) + 2000);
                                while (publish < maxPublish) {
                                    SmartData smartData = getRandomSmartData(unit, threadNum, publish);

                                    startTransaction = System.currentTimeMillis();
                                    transaction = contract.createTransaction("publishSensorDataTestContext");
                                    transaction.submit(sensorFullName, Byte.toString(smartData.version),
                                            Long.toString(smartData.unit), Long.toString(smartData.timestamp),
                                            Double.toString(smartData.value), Byte.toString(smartData.error),
                                            Byte.toString(smartData.confidence), Integer.toString(smartData.dev));
                                    transactionTimeWait += System.currentTimeMillis() - startTransaction;

                                    // System.out.println(new String(transactionResult, StandardCharsets.UTF_8));
                                    publish++;
                                    Thread.sleep(interval);
                                }
                            } catch (Exception e) {
                                System.out.println("Exception in thread " + Integer.toString(threadNum));
                                e.printStackTrace();
                            }
                            totalExecutionTime = System.currentTimeMillis() - startExecution;

                            // signature time testing
                            TransactionContext txContext = transaction.getTransactionContext();
                            try {
                                startSignature = System.currentTimeMillis();
                                txContext.sign("EAGYEASDIUHWAUIHDIASDdsaUSAHDIUADHUIWH".getBytes());
                                singleSignatureTime = System.currentTimeMillis() - startSignature;
                            } catch (CryptoException | InvalidArgumentException e) {
                                e.printStackTrace();
                            }

                            System.out.println(getClass().getName() + " Thread " + Integer.toString(threadNum)
                                    + " took " + Long.toString(transactionTimeWait) + "ms to submit "
                                    + Integer.toString(maxPublish) + " transactions of "
                                    + Long.toString(totalExecutionTime)
                                    + "ms total execution time. \nA single signature takes: "
                                    + Long.toString(singleSignatureTime) + "ms ");

                            // Evaluate transactions that query state from the ledger.
                            // byte[] queryResponse = contract.evaluateTransaction("query", "A");
                            // System.out.println(new String(queryResponse, StandardCharsets.UTF_8));

                        } catch (Exception e) {// (ContractException e) {
                            e.printStackTrace();
                        }
                    }
                };
                threads[i].start();
            }

            for (int i = 1; i <= THREAD_NUM; i++)
                threads[i].join();

            System.out.println("ENDED!");
            System.exit(0);
        }
    }
}