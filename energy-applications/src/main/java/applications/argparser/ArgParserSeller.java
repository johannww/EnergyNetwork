package applications.argparser;

import org.apache.commons.cli.Option;

public class ArgParserSeller extends ArgParser {

    @Override
    protected void addSpecificOptions(String[] args) {
        programName = "Seller.jar";
        header = "\n1) ";

        Option sellerCertPath = new Option("cert", "certificate", true, "path to the seller x509 MSP certificate");
        options.addOption(sellerCertPath);

        Option sellerPkPath = new Option("pk", "privatekey", true, "path to the seller private key");
        options.addOption(sellerPkPath);

        Option sell = new Option("s", "sell", true, "SELL ENERGY - indicates that the a SELLING must be performed");
        sell.setArgs(0);
        options.addOption(sell);

        Option energyAmountKWH = new Option("kwh", "energyquantitykwh", true,
                "SELL ENERGY - energy quantity in Kilowatt-hour to be sold");
        options.addOption(energyAmountKWH);

        Option pricePerKWH = new Option("price", "priceperkwh", true,
                "SELL ENERGY - the amount to be paid per Kilowatt-hour");
        options.addOption(pricePerKWH);

        Option energyType = new Option("type", "energytype", true, "desired energy type to be bought");
        options.addOption(energyType);

        Option proveToUtility = new Option("prove", "proveacquisition", true,
                "indicates the seller wants to prove to his utility company the energy he sold through the Energy Network");
        proveToUtility.setArgs(0);
        options.addOption(proveToUtility);
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
                throw new Error(
                        "Argument '--enroll' cannot be passed with '--certificate' or '--privatekey'");

        } else if (!cmd.hasOption("certificate") && !cmd.hasOption("privatekey")
                && !cmd.hasOption("walletpath"))
            throw new Error(
                    "At least one set of flags '--enroll', ('--certificate' with '--privatekey') or '--walletpath' must be passed");

        if (cmd.hasOption("walletpath"))
            if (!cmd.hasOption("user"))
                throw new Error("flag '--user' must be passed with '--wallet'");

        if (cmd.hasOption("sell")) {
            if (!cmd.hasOption("energyquantitykwh"))
                throw new Error("Flag '--energyquantitykwh' must be passed to SELL ENERGY");
            if (!cmd.hasOption("priceperkwh"))
                throw new Error("Flag '--priceperkwh' must be passed to SELL ENERGY");
            if (!cmd.hasOption("energytype"))
                throw new Error("Flag '--energytype' must be passed to SELL ENERGY");
        }
    }

    // verify prove to utility company

}