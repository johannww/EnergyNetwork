package applications.testargparser;

import org.apache.commons.cli.CommandLine;
import org.apache.commons.cli.CommandLineParser;
import org.apache.commons.cli.DefaultParser;
import org.apache.commons.cli.HelpFormatter;
import org.apache.commons.cli.Option;
import org.apache.commons.cli.Options;
import org.apache.commons.cli.ParseException;

public class ArgParserSellerTest {

    protected CommandLine cmd;
    protected Options options;

    public ArgParserSellerTest() {
        options = new Options();
    }

    public CommandLine parseArgs(String[] args) throws Exception {
        Option help = new Option("h", "help", true, "help");
        help.setArgs(0);
        options.addOption(help);

        Option mspId = new Option("msp", "membership", true, "case sensitive buyer's organization MSP ID");
        mspId.setRequired(true);
        options.addOption(mspId);

        Option baseDir = new Option("basedir", "basedir", true, "EnergyNetwork root dir");
        mspId.setRequired(true);
        options.addOption(baseDir);

        Option buyerQuantity = new Option("sellers", "sellers", true,
                "For each sensor we create 1 threat to publish 'publishquantity' times a SmartData of unit 'unit' every 'publishinterval' interval.");
        buyerQuantity.setRequired(true);
        options.addOption(buyerQuantity);

        Option interval = new Option("publishinterval", "publishinterval", true,
                "Interval IN MILLI SECONDS between two BuyBid WHOLE publishes");
        interval.setRequired(true);
        options.addOption(interval);

        Option buyBidQuantity = new Option("publishquantity", "publishquantity", true,
                "Number of BuyBid to be sent on TESTING!");
        buyBidQuantity.setRequired(true);
        options.addOption(buyBidQuantity);

        Option paymentUrl = new Option("paymentcompanyurl", "paymentcompanyurl", true,
        "URL for interacting with the Payment Company!");
        paymentUrl.setRequired(true);
        options.addOption(paymentUrl);

        Option inDockerPrivateNetwork = new Option("dockernetwork", "dockernetwork", true,
        "Flag to infor to test application that it will be run inside the docker private network to fetch the correct 'connection-tls.json'");
        inDockerPrivateNetwork.setArgs(0);
        options.addOption(inDockerPrivateNetwork);

        CommandLineParser parser = new DefaultParser();
        HelpFormatter formatter = new HelpFormatter();
        formatter.setLongOptSeparator("\n");
        String header = "\n1) Test seller";
        cmd = null;

        try {
            cmd = parser.parse(options, args);
        } catch (ParseException e) {
            System.out.println(e.getMessage());
            formatter.printHelp("SellerForTest.jar", header, options, "");
            System.exit(1);
        }

        // print help if the 'help' option is present
        if (cmd.hasOption("help")) {
            formatter.printHelp("Buyer.jar", header, options, "");
            System.exit(1);
        }

        return cmd;
    }

}