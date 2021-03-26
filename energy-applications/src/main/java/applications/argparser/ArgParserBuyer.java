package applications.argparser;


import org.apache.commons.cli.Option;


public class ArgParserBuyer extends ArgParser {

    @Override
    protected void addSpecificOptions(String[] args) {
        programName = "Buyer.jar";

        header = "\n1) Buy energy on the Hyperledger Energy Network.\n2) The IDEMIX credentials might be loaded through ENROLLMENT, FILESYSTEM and GATEWAY WALLET.\n3)You can also prove to your Utility Company the energy you bought, in order to get bill discounts\n";

        Option mspFolder = new Option("cp", "credentialspath", true,
                "path to the user folder with the subfolders 'msp' and 'user', containing the IDEMIX Credentials. Expected folder structure: https://hyperledger-fabric.readthedocs.io/en/release-2.2/idemixgen.html?highlight=idemix#directory-structure");
        options.addOption(mspFolder);

        Option paymentCompanyID = new Option("pci", "paymentcompanyid", true,
                "BUY ENERGY - Payment Company MSP ID on the Energy Network");
        options.addOption(paymentCompanyID);

        Option paymentToken = new Option("t", "token", true,
                "BUY ENERGY - the token provided by the Payment Company to the energy buyer");
        options.addOption(paymentToken);

        Option energyAmountKWH = new Option("kwh", "energyquantitykwh", true,
                "BUY ENERGY - energy quantity in Kilowatt-hour to be bought");
        options.addOption(energyAmountKWH);

        Option pricePerKWH = new Option("price", "priceperkwh", true,
                "BUY ENERGY - the amount to be paid per Kilowatt-hour");
        options.addOption(pricePerKWH);

        Option energyType = new Option("type", "energytype", true, "desired energy type to be bought");
        options.addOption(energyType);

        Option proveToUtility = new Option("prove", "proveacquisition", true,
                "indicates the buyer wants to prove to his utility company the energy he bought through the Energy Network");
        proveToUtility.setArgs(0);
        options.addOption(proveToUtility);

        Option utilityProofHost = new Option("ua", "utilityaddress", true,
                "the the utility address (e.g. https://energycompany.com/prove) for the ");
        utilityProofHost.setArgs(0);
        options.addOption(utilityProofHost);
    }

    @Override
    protected void checkSpecific() {
        // check if user passed the flags from where to load and save the IDEMIX
        // credentials
        if (cmd.hasOption("enroll")) {
            if (!cmd.hasOption("credentialspath")) {
                if (!cmd.hasOption("user"))
                    throw new Error("Flag '--user' must be passed with 'enroll'");
                if (!cmd.hasOption("password"))
                    throw new Error("Flag '--password' must be passed with 'enroll'");
                if (!cmd.hasOption("host"))
                    throw new Error("Flag '-host' must be passed with 'enroll'");
                if (!cmd.hasOption("cacert"))
                    throw new Error("Flag '--cacert' must be passed with 'enroll'");
            } else
                throw new Error("Argument '--enroll' cannot be passed with '--credentialspath'");

        } else if (!cmd.hasOption("credentialspath") && !cmd.hasOption("walletpath"))
            throw new Error(
                    "At least one of the flags '--enroll', '--credentialspath' or '--walletpath' must be passed");

        if (cmd.hasOption("walletpath"))
            if (!cmd.hasOption("user"))
                throw new Error("flag '--user' must be passed with '--wallet'");
        if (cmd.hasOption("paymentcompanyid") || cmd.hasOption("token") || cmd.hasOption("energyquantitykwh")
                || cmd.hasOption("priceperkwh") || cmd.hasOption("energytype")) {
            if (!cmd.hasOption("paymentcompanyid"))
                throw new Error("Flag '--paymentcompanyid' must be passed to BUY ENERGY");
            if (!cmd.hasOption("token"))
                throw new Error("Flag '--token' must be passed to BUY ENERGY");
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