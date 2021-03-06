package applications;

import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.function.Consumer;

import org.apache.commons.cli.CommandLine;

import org.hyperledger.fabric.gateway.Contract;
import org.hyperledger.fabric.gateway.ContractEvent;
import org.hyperledger.fabric.gateway.Gateway;
import org.hyperledger.fabric.gateway.Identity;
import org.hyperledger.fabric.gateway.Network;
import org.hyperledger.fabric.gateway.Transaction;


import applications.argparser.ArgParserPeriodicAuction;
import applications.identity.ApplicationIdentityProvider;


public class AppPeriodicAuction {

    private static CommandLine cmd;
    private static Network network;
    private static int failCount = 0;
    private static int auctionCount = 0;



    private static void registerAuctionEventListener(Contract contract, Long auctionInterval) {
        Consumer<ContractEvent> auctionPerfomedListener = new Consumer<ContractEvent>() {

            @Override
            public void accept(ContractEvent t) {

                if (t.getName().equals("auctionPerformed")) {
                    try {
                        Thread.sleep(auctionInterval);
                    } catch (Exception e) {
                        System.out.println("Exception in timer");
                    }
                    Transaction transaction = contract.createTransaction("auction");
                    try {
                        auctionCount++;
                        transaction.submit();
                        System.out.printf("Auction transaction %d was SUCCESSFULLY submited to the orderer\n", auctionCount);
                    } catch (Exception e) {
                        failCount++;
                        System.out.printf("Auction transaction %d commit failed for the %d time\n", auctionCount, failCount);
                        System.out.println("The transaction will probably appear in following blocks");
                        System.out.println(e.getMessage());
                    }
                }
            }
        };
        contract.addContractListener(auctionPerfomedListener, "auctionPerformed");
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
                "D:\\UFSC\\Mestrado\\Hyperledger\\Fabric\\EnergyNetwork\\hyperledger\\ufsc\\admin1\\msp\\keystore\\key.pem", "-msp",
                "UFSC", "--auctioninterval", "100000" };*/

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
        Gateway.Builder builder = Gateway.createBuilder().identity(identity).networkConfig(networkConfigFile).discovery((dockerPrefix.length() > 0) || (awsPrefix.length() > 0));
        

        // publishing the buybid
        // Create a gateway connection
        try (Gateway gateway = builder.connect()){


            // Obtain a smart contract deployed on the network.
            network = gateway.getNetwork("canal");
            Contract contract = network.getContract("energy");

            Long auctionInterval = Long.parseLong(cmd.getOptionValue("auctioninterval"));
            registerAuctionEventListener(contract, auctionInterval);

            Transaction transaction = contract.createTransaction("auction");
            try {
                auctionCount++;
                transaction.submit();
                System.out.printf("Auction transaction %d was SUCCESSFULLY submited to the orderer\n", auctionCount);
            } catch (Exception e) {
                failCount++;
                System.out.printf("Auction transaction %d commit failed for the %d time\n", auctionCount, failCount);
                System.out.println("The transaction will probably appear in following blocks");
                System.out.println(e.getMessage());
            }
            
            Thread.currentThread().join();


        } catch (Exception e) {
            e.printStackTrace();
        }

        // save SOMEHOW the idemix params for proving the buybid to the utility company

    }

}
