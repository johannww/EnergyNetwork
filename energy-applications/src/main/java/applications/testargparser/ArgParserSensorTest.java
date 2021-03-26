package applications.testargparser;

import org.apache.commons.cli.CommandLine;
import org.apache.commons.cli.CommandLineParser;
import org.apache.commons.cli.DefaultParser;
import org.apache.commons.cli.HelpFormatter;
import org.apache.commons.cli.Option;
import org.apache.commons.cli.Options;
import org.apache.commons.cli.ParseException;

public class ArgParserSensorTest {

    protected CommandLine cmd;
    protected Options options;

    public ArgParserSensorTest() {
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

        Option sensorQuantity = new Option("sensors", "sensors", true,
                "For each sensor we create 1 threat to publish 'publishquantity' times a SmartData of unit 'unit' every 'publishinterval' interval.");
        sensorQuantity.setRequired(true);
        options.addOption(sensorQuantity);

        Option unit = new Option("unit", "unit", true,
                "Long representing Smart Data unit https://epos.lisha.ufsc.br/EPOS+2+User+Guide#Unit");
        unit.setRequired(true);
        options.addOption(unit);

        Option interval = new Option("publishinterval", "publishinterval", true,
                "Interval IN MILLI SECONDS between two SmartData publishes");
        interval.setRequired(true);
        options.addOption(interval);

        Option smartDataQuantity = new Option("publishquantity", "publishquantity", true,
                "Number of SmartData to be sent on TESTING!");
        smartDataQuantity.setRequired(true);
        options.addOption(smartDataQuantity);

        Option inDockerPrivateNetwork = new Option("dockernetwork", "dockernetwork", true,
        "Flag to infor to test application that it will be run inside the docker private network to fetch the correct 'connection-tls.json'");
        inDockerPrivateNetwork.setArgs(0);
        options.addOption(inDockerPrivateNetwork);

        CommandLineParser parser = new DefaultParser();
        HelpFormatter formatter = new HelpFormatter();
        formatter.setLongOptSeparator("\n");
        String header = "\n1) Test the Sensors";
        cmd = null;

        try {
            cmd = parser.parse(options, args);
        } catch (ParseException e) {
            System.out.println(e.getMessage());
            formatter.printHelp("Buyer.jar", header, options, "");
            System.exit(1);
        }

        // print help if the 'help' option is present
        if (cmd.hasOption("help")) {
            formatter.printHelp("SensorForTest.jar", header, options, "");
            System.exit(1);
        }

        return cmd;
    }

}
