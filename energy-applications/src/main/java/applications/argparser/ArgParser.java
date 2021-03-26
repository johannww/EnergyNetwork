package applications.argparser;

import org.apache.commons.cli.CommandLine;
import org.apache.commons.cli.CommandLineParser;
import org.apache.commons.cli.DefaultParser;
import org.apache.commons.cli.HelpFormatter;
import org.apache.commons.cli.Option;
import org.apache.commons.cli.Options;
import org.apache.commons.cli.ParseException;

public abstract class ArgParser {
    
    protected CommandLine cmd;
    protected Options options;
    protected String header;
    protected String programName;

    ArgParser(){
        options = new Options();
        header = "";
        programName = "";
    }


    public CommandLine parseArgs(String[] args) throws Exception {
        Option help = new Option("h", "help", true, "help");
        help.setArgs(0);
        options.addOption(help);

        Option enroll = new Option("e", "enroll", true,
                "ENROLL - indicates that the enrollment with the CA must be performed with the provided '--user', '--password', '--cahost' and '--cacert'");
        enroll.setArgs(0);
        options.addOption(enroll);

        Option enrollUser = new Option("u", "user", true,
                "ENROLL - user registered in the 'CA' to be enrolled or used as identity label to load the wallet");
        options.addOption(enrollUser);

        Option enrollPassword = new Option("pw", "password", true,
                "ENROLL - password of the user registered in the 'CA' to be enrolled");
        options.addOption(enrollPassword);

        Option caHostAndPort = new Option("host", "host", true,
                "ENROLL - certificate authority host on format: 'https://cahost:port' or 'http://cahost:port'");
        options.addOption(caHostAndPort);

        Option caPemCertPath = new Option("cacert", "cacert", true,
                "ENROLL - path to the root CA's PEM certificate for the buyer organization");
        options.addOption(caPemCertPath);

        Option walletPath = new Option("w", "walletpath", true,
                "path to save to or load from the wallet contaning the IDEMIX credentials");
        options.addOption(walletPath);

        Option mspId = new Option("msp", "membership", true, "case sensitive buyer's organization MSP ID");
        mspId.setRequired(true);
        options.addOption(mspId);

        Option inDockerPrivateNetwork = new Option("dockernetwork", "dockernetwork", true,
        "Flag to infor to test application that it will be run inside the docker private network to fetch the correct 'connection-tls.json'");
        inDockerPrivateNetwork.setArgs(0);
        options.addOption(inDockerPrivateNetwork);
        
        // take more args
        addSpecificOptions(args);

        CommandLineParser parser = new DefaultParser();
        HelpFormatter formatter = new HelpFormatter();
        formatter.setLongOptSeparator("\n");
        cmd = null;

        try {
            cmd = parser.parse(options, args);
        } catch (ParseException e) {
            System.out.println(e.getMessage());
            formatter.printHelp(programName, header, options, "");
            System.exit(1);
        }

        // print help if the 'help' option is present
        if (cmd.hasOption("help")) {
            formatter.printHelp("Buyer.jar", header, options, "");
            System.exit(1);
        }

        checkSpecific();
        return cmd;
    }

    protected abstract void addSpecificOptions(String[] args);

    protected abstract void checkSpecific();

}