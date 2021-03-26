package applications.argparser;

import org.apache.commons.cli.Option;

public class ArgParserSensor extends ArgParser {

    @Override
    protected void addSpecificOptions(String[] args) {
        programName = "Sensor.jar";
        header = "\n1) ";

        Option sensorCertPath = new Option("cert", "certificate", true, "path to the sensor x509 MSP certificate");
        options.addOption(sensorCertPath);

        Option sensorPkPath = new Option("pk", "privatekey", true, "path to the sensor private key");
        options.addOption(sensorPkPath);

        Option unit = new Option("unit", "unit", true,
                "Long representing Smart Data unit https://epos.lisha.ufsc.br/EPOS+2+User+Guide#Unit");
        options.addOption(unit);

        Option interval = new Option("publishinterval", "publishinterval", true,
                "Interval IN SECONDS between two SmartData publishes");
        options.addOption(interval);

        Option smartDataQuantity = new Option("publishquantity", "publishquantity", true,
                "Number of SmartData to be sent on TESTING!");
        options.addOption(smartDataQuantity);
    }

    @Override
    protected void checkSpecific() {
        // check if user passed the flags from where to load and save the certificates
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