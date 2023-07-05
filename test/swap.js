const Fury = require('@kava-labs/javascript-sdk');
const { sleep } = require("./helpers.js");

const incomingSwap = async (furyClient, bnbClient, assets, denom, amount) => {
  const assetInfo = assets[denom];
  if(!assetInfo) {
      throw new Error(denom + " is not supported by nmtool BEP3");
  }

  // Assets involved in the swap
  const asset = assetInfo.binanceChainDenom;

  // Addresses involved in the swap
  const sender = bnbClient.getClientKeyAddress();
  const recipient = assetInfo.binanceChainDeputyHotWallet; // deputy's address on Binance Chain
  const senderOtherChain = assetInfo.furyDeputyHotWallet; // deputy's address on Fury
  const recipientOtherChain = furyClient.wallet.address;

  // Format asset/amount parameters as tokens, expectedIncome
  const tokens = [
    {
      denom: asset,
      amount: amount,
    },
  ];
  const expectedIncome = [String(amount), ':', asset].join('');

  // Number of blocks that swap will be active
  const heightSpan = 10001;

  // Generate random number hash from timestamp and hex-encoded random number
  let randomNumber = Fury.utils.generateRandomNumber();
  const timestamp = Math.floor(Date.now() / 1000);
  const randomNumberHash = Fury.utils.calculateRandomNumberHash(
    randomNumber,
    timestamp
  );
  console.log('Secret random number:', randomNumber);

  const swapIDs = calcSwapIDs(randomNumberHash, sender, senderOtherChain);
  console.log('Expected Binance Chain swap ID:', swapIDs.origin);

  // Send create swap tx using Binance Chain client
  console.log('\nRaw transaction data:')
  const res = await bnbClient.swap.HTLT(
    sender,
    recipient,
    recipientOtherChain,
    senderOtherChain,
    randomNumberHash,
    timestamp,
    tokens,
    expectedIncome,
    heightSpan,
    true
  );

  if (res && res.status == 200) {
    console.log('\nCreate swap tx hash (Binance Chain): ', res.result.result.hash);
  } else {
    console.log('Tx error:', res);
    return;
  }
  // Wait for deputy to see the new swap on Binance Chain and relay it to Fury
  console.log('Waiting for deputy to witness and relay the swap...');
  console.log('Expected Fury swap ID:', swapIDs.dest);

  await sleep(45000); // 45 seconds
  await furyClient.getSwap(swapIDs.dest);

  // Send claim swap tx using Fury client
  const txHashClaim = await furyClient.claimSwap(
    swapIDs.dest,
    randomNumber
  );
  console.log('Claim swap tx hash (Fury): '.concat(txHashClaim));

  // Check the claim tx hash
  const txRes = await furyClient.checkTxHash(txHashClaim, 15000);
  console.log('\nTx result:', txRes.raw_log);
};

const outgoingSwap = async(furyClient, bnbClient, assets, denom, amount) => {
  const assetInfo = assets[denom];
  if(!assetInfo) {
    throw new Error(denom + " is not supported by nmtool BEP3");
  }

  const sender = furyClient.wallet.address;
  const recipient = assetInfo.furyDeputyHotWallet; // deputy's address on nemo
  const senderOtherChain = assetInfo.binanceChainDeputyHotWallet; // deputy's address on bnbchain
  const recipientOtherChain = bnbClient.getClientKeyAddress();

  // Set up params
  const asset = assetInfo.furyDenom;

  const coins = Fury.utils.formatCoins(amount, asset);
  const heightSpan = "250";

  // Generate random number hash from timestamp and hex-encoded random number
  const randomNumber = Fury.utils.generateRandomNumber();
  const timestamp = Math.floor(Date.now() / 1000);
  const randomNumberHash = Fury.utils.calculateRandomNumberHash(
    randomNumber,
    timestamp
  );
  console.log("\nSecret random number:", randomNumber.toUpperCase());

  const swapIDs = calcSwapIDs(randomNumberHash, sender, senderOtherChain);
  console.log('Expected Fury swap ID:', swapIDs.origin);

  const txHash = await furyClient.createSwap(
    recipient,
    recipientOtherChain,
    senderOtherChain,
    randomNumberHash,
    timestamp,
    coins,
    heightSpan
  );

  console.log("\nTx hash (Create swap on Fury):", txHash);

  // Wait for deputy to see the new swap on Fury and relay it to Binance Chain
  console.log("Waiting for deputy to witness and relay the swap...")
  console.log('Expected Binance Chain swap ID:', swapIDs.dest);
  await sleep(45000); // 45 seconds


  console.log('\nRaw transaction data:')
  const res = await bnbClient.swap.claimHTLT(bnbClient.getClientKeyAddress(), swapIDs.dest, randomNumber);
  if (res && res.status == 200) {
    console.log(
      "\nClaim swap tx hash (Binance Chain): ",
      res.result.result.hash
    );
  } else {
    console.log("Tx error:", res);
    return;
  }
};

// Print swap IDs
var calcSwapIDs = (randomNumberHash, sender, senderOtherChain) => {
  // Calculate the expected swap ID on origin chain
  const originChainSwapID = Fury.utils.calculateSwapID(
    randomNumberHash,
    sender,
    senderOtherChain
  );

  // Calculate the expected swap ID on destination chain
  const destChainSwapID = Fury.utils.calculateSwapID(
    randomNumberHash,
    senderOtherChain,
    sender
  );

  return { origin: originChainSwapID, dest: destChainSwapID }
};

module.exports = {
    incomingSwap,
    outgoingSwap
}
