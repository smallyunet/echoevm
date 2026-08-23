"use strict";

document.getElementById("open-etherscan").addEventListener("click", () => {
  chrome.tabs.create({ url: "https://etherscan.io/contractsVerified" });
});
