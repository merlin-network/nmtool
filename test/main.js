const {FURY_ENDPOINT_NMTOOL, BINANCE_CHAIN_ENDPOINT_NMTOOL, LOADED_FURY_MFURYNIC,
    LOADED_BINANCE_CHAIN_MFURYNIC, BEP3_ASSETS } = require("./config.js");
const { setup, loadFuryDeputies } = require("./nmtool.js");
const { incomingSwap, outgoingSwap } = require("./swap.js");

var main = async () => {
    // Initialize clients compatible with nmtool
    const clients = await setup(FURY_ENDPOINT_NMTOOL, BINANCE_CHAIN_ENDPOINT_NMTOOL,
        LOADED_FURY_MFURYNIC, LOADED_BINANCE_CHAIN_MFURYNIC);

    // Load each Fury deputy hot wallet
    await loadFuryDeputies(clients.furyClient, BEP3_ASSETS, 100000);

    await incomingSwap(clients.furyClient, clients.bnbClient, BEP3_ASSETS, "busd", 10200005);
    // await outgoingSwap(clients.furyClient, clients.bnbClient, BEP3_ASSETS, "busd", 500005);
};

main();
