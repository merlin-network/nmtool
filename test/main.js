const {NEMO_ENDPOINT_NMTOOL, BINANCE_CHAIN_ENDPOINT_NMTOOL, LOADED_NEMO_MNEMONIC,
    LOADED_BINANCE_CHAIN_MNEMONIC, BEP3_ASSETS } = require("./config.js");
const { setup, loadNemoDeputies } = require("./nmtool.js");
const { incomingSwap, outgoingSwap } = require("./swap.js");

var main = async () => {
    // Initialize clients compatible with nmtool
    const clients = await setup(NEMO_ENDPOINT_NMTOOL, BINANCE_CHAIN_ENDPOINT_NMTOOL,
        LOADED_NEMO_MNEMONIC, LOADED_BINANCE_CHAIN_MNEMONIC);

    // Load each Nemo deputy hot wallet
    await loadNemoDeputies(clients.nemoClient, BEP3_ASSETS, 100000);

    await incomingSwap(clients.nemoClient, clients.bnbClient, BEP3_ASSETS, "busd", 10200005);
    // await outgoingSwap(clients.nemoClient, clients.bnbClient, BEP3_ASSETS, "busd", 500005);
};

main();
