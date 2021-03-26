package applications.argparser;

import org.apache.commons.cli.Option;

public class ArgParserUtility extends ArgParser {

    @Override
    protected void addSpecificOptions(String[] args) {
        programName = "Utility.jar";
        header = "\n1) ";

        Option utilityCertPath = new Option("cert", "certificate", true, "path to the utility x509 MSP certificate");
        options.addOption(utilityCertPath);

        Option utilityPkPath = new Option("pk", "privatekey", true, "path to the utility private key");
        options.addOption(utilityPkPath);

        Option httpPort = new Option("port", "port", true, "port for the HTTP server to listen");
        httpPort.setRequired(true);
        options.addOption(httpPort);
    }

    @Override
    protected void checkSpecific() {
        // check if user passed the flags from where to load and save the credentials
        if (cmd.hasOption("enroll")) {
            if (!cmd.hasOption("certificate") && !cmd.hasOption("privatekey")) {
                if (!cmd.hasOption("user"))
                    throw new Error("Flag '--user' must be passed with 'enroll'");
                if (!cmd.hasOption("password"))
                    throw new Error("Flag '--password' must be passed with 'enroll'");
                if (!cmd.hasOption("host"))
                    throw new Error("Flag '-host' must be passed with 'enroll'");
                if (!cmd.hasOption("cacert"))
                    throw new Error("Flag '--cacert' must be passed with 'enroll'");
            } else
                throw new Error("Argument '--enroll' cannot be passed with '--certificate' or '--privatekey'");

        } else if (!cmd.hasOption("certificate") && !cmd.hasOption("privatekey") && !cmd.hasOption("walletpath"))
            throw new Error(
                    "At least one set of flags '--enroll', ('--certificate' with '--privatekey') or '--walletpath' must be passed");

        if (cmd.hasOption("walletpath"))
            if (!cmd.hasOption("user"))
                throw new Error("flag '--user' must be passed with '--wallet'");
    }
}