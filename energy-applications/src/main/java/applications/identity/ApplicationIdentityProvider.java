package applications.identity;

import java.io.IOException;
import java.io.Reader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.InvalidKeyException;
import java.security.PrivateKey;
import java.security.cert.CertificateException;
import java.security.cert.X509Certificate;
import java.util.Properties;

import org.apache.commons.cli.CommandLine;
import org.hyperledger.fabric.gateway.IdemixIdentity;
import org.hyperledger.fabric.gateway.Identities;
import org.hyperledger.fabric.gateway.Identity;
import org.hyperledger.fabric.gateway.Wallet;
import org.hyperledger.fabric.gateway.Wallets;
import org.hyperledger.fabric.sdk.Enrollment;
import org.hyperledger.fabric.sdk.identity.IdemixEnrollment;
import org.hyperledger.fabric.sdk.security.CryptoSuite;
import org.hyperledger.fabric_ca.sdk.HFCAClient;

public final class ApplicationIdentityProvider {
    
    private static X509Certificate readX509Certificate(final Path certificatePath)
            throws IOException, CertificateException {
        try (Reader certificateReader = Files.newBufferedReader(certificatePath, StandardCharsets.UTF_8)) {
            return Identities.readX509Certificate(certificateReader);
        }
    }

    private static PrivateKey getPrivateKey(final Path privateKeyPath) throws IOException, InvalidKeyException {
        try (Reader privateKeyReader = Files.newBufferedReader(privateKeyPath, StandardCharsets.UTF_8)) {
            return Identities.readPrivateKey(privateKeyReader);
        }
    }

    public static Identity getX509Identity(CommandLine cmd) throws Exception {
        Identity identity = null;
        // enroll, load certificate files or load certificate wallet
        if (cmd.hasOption("enroll")) {

            String enrollUser = cmd.getOptionValue("user");
            String enrollPassword = cmd.getOptionValue("password");
            String caHostAndPort = cmd.getOptionValue("host");

            Path caCertPath = Paths.get(cmd.getOptionValue("cacert"));
            String caCertPemStr = Files.readString(caCertPath);

            String mspId = cmd.getOptionValue("msp");

            Properties props = new Properties();
            props.put("pemBytes", caCertPemStr.getBytes());
            props.put("allowAllHostNames", "true");
            HFCAClient ca = HFCAClient.createNewInstance(caHostAndPort, props);
            CryptoSuite cryptoSuite = CryptoSuite.Factory.getCryptoSuite();
            ca.setCryptoSuite(cryptoSuite);

            Enrollment enrollment = ca.enroll(enrollUser, enrollPassword);
            // generating new Identity with the enrollment information
            identity = Identities.newX509Identity(mspId, enrollment);

            if (cmd.hasOption("walletpath")) {
                // saving enrollment in wallet
                Path walletPath = Paths.get(cmd.getOptionValue("walletpath"));
                Wallet wallet = Wallets.newFileSystemWallet(walletPath);
                String identityLabel = cmd.getOptionValue("user");

                wallet.put(identityLabel, identity);
            }

        } else if (cmd.hasOption("certificate")) {

            String mspId = cmd.getOptionValue("msp");
            // Path to certificate and private key
            Path certificatePath = Paths.get(cmd.getOptionValue("certificate"));
            Path privateKeyPath = Paths.get(cmd.getOptionValue("privatekey"));
            // Creating Identity
            X509Certificate cert = readX509Certificate(certificatePath);
            PrivateKey pk = getPrivateKey(privateKeyPath);
            identity = Identities.newX509Identity(mspId, cert, pk);

            if (cmd.hasOption("walletpath")) {
                // saving enrollment in wallet
                Path walletPath = Paths.get(cmd.getOptionValue("walletpath"));
                Wallet wallet = Wallets.newFileSystemWallet(walletPath);
                String identityLabel = cmd.getOptionValue("user");

                wallet.put(identityLabel, identity);
            }

        } else if (cmd.hasOption("walletpath")) {

            Path walletPath = Paths.get(cmd.getOptionValue("walletpath"));
            Wallet wallet = Wallets.newFileSystemWallet(walletPath);
            String identityLabel = cmd.getOptionValue("user");

            // Loading an existing wallet holding identities used to access the network.
            identity = wallet.get(identityLabel);

        } else {
            throw new Exception("No identity found!");
        }
        return identity;
    }

    public static IdemixIdentity  getIdemixIdentity(CommandLine cmd) throws Exception {
        IdemixIdentity idemixId = null;
        // enroll, load idemix files or load idemix wallet
        if (cmd.hasOption("enroll")) {

            String enrollUser = cmd.getOptionValue("user");
            String enrollPassword = cmd.getOptionValue("password");
            String caHostAndPort = cmd.getOptionValue("host");

            Path caCertPath = Paths.get(cmd.getOptionValue("cacert"));
            String caCertPemStr = Files.readString(caCertPath);

            String mspId = cmd.getOptionValue("msp");

            Properties props = new Properties();
            props.put("pemBytes", caCertPemStr.getBytes());
            props.put("allowAllHostNames", "true");
            HFCAClient ca = HFCAClient.createNewInstance(caHostAndPort, props);
            CryptoSuite cryptoSuite = CryptoSuite.Factory.getCryptoSuite();
            ca.setCryptoSuite(cryptoSuite);

            Enrollment firstEnrollment = ca.enroll(enrollUser, enrollPassword);
            IdemixEnrollment enrollment = (IdemixEnrollment) ca.idemixEnroll(firstEnrollment, mspId);
            // generating new Idemix Identity with the enrollment information
            idemixId = Identities.newIdemixIdentity(mspId, enrollment);

            if (cmd.hasOption("walletpath")) {
                // saving enrollment in wallet
                Path walletPath = Paths.get(cmd.getOptionValue("walletpath"));
                Wallet wallet = Wallets.newFileSystemWallet(walletPath);
                String identityLabel = cmd.getOptionValue("user");

                wallet.put(identityLabel, idemixId);
            }

        } else if (cmd.hasOption("credentialspath")) {

            String mspId = cmd.getOptionValue("msp");
            // Path to idemix credentials context: IssuerPublicKey, RevocationPublicKey and
            // SignerConfig
            Path pathIpk = Paths.get(cmd.getOptionValue("credentialspath"), "msp", "IssuerPublicKey");
            Path pathRpk = Paths.get(cmd.getOptionValue("credentialspath"), "msp", "RevocationPublicKey");
            Path pathSignerCofing = Paths.get(cmd.getOptionValue("credentialspath"), "user", "SignerConfig.json");
            // Creating Idemix Identity
            idemixId = Identities.newIdemixIdentity(mspId, pathIpk, pathRpk, pathSignerCofing);

            if (cmd.hasOption("walletpath")) {
                // saving enrollment in wallet
                Path walletPath = Paths.get(cmd.getOptionValue("walletpath"));
                Wallet wallet = Wallets.newFileSystemWallet(walletPath);
                String identityLabel = cmd.getOptionValue("user");

                wallet.put(identityLabel, idemixId);
            }

        } else if (cmd.hasOption("walletpath")) {

            Path walletPath = Paths.get(cmd.getOptionValue("walletpath"));
            Wallet wallet = Wallets.newFileSystemWallet(walletPath);
            String identityLabel = cmd.getOptionValue("user");

            // Loading an existing wallet holding identities used to access the network.
            idemixId = (IdemixIdentity) wallet.get(identityLabel);
        } else {
            throw new Exception("No identity found!");
        }
        return idemixId;
    }
}