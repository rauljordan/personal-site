---
layout: post
title:  "How to Scale Ethereum: Sharding Explained"
preview: The scalability debate is at the front and center of the crypto community. With major occurrences such as the Cryptokitties debacle clogging up the entire Ethereum network over the span of a few days, it is well-known that the biggest, public blockchains in their current state do not scale...
date: 2018-Jan-10
tags: 
  - blockchain
---

The scalability debate is at the front and center of the crypto community. With major occurrences such as the Cryptokitties debacle clogging up the entire Ethereum network over the span of a few days, it is well-known that the biggest, public blockchains in their current state do not scale.

So what are the approaches the community has decided to take? The solution is two-fold. The first approach is to improve scaling through off-chain solutions, also known as layer-2 scaling, where some transactions are handled off the blockchain and only interact with it sparingly.  The other approach is to modify the design of the protocol altogether to fix the fundamental problems with parallelizability the blockchain faces. Unfortunately, many of us protocol devs often look at these problems and instantly feel put off by the immense difficulty they pose.

Although we’re still in the early stages of Ethereum, the community is filled with some of the smartest minds in tech, with so many innovations happening at breakneck speed. It’s easy to feel that there are smarter devs out there that are probably way more qualified to tackle monumental problems such as scalability, but this feeling is what’s holding us back. Truth is, the community is willing and ready to help anyone who wants to get involved, and yes that includes you! This post will break down the current approach the Ethereum core team is taking towards sharding and expose its current limitations and paths for improvement. By the end of this post, you’ll know enough to explore this problem on your own and who knows, maybe you’ll be the one to build the first sharding client!

![image](https://i.imgur.com/VIIZfQ6.png)

As the number of transactions on Ethereum keeps going up and up, we have no time to lose. Let’s get started.

## What is Sharding?

Currently, every single node running the Ethereum network has to process every single transaction that goes through the network. This gives the blockchain a high amount of security because of how much validation goes into each block, but at the same time it means that an entire blockchain is only as fast as its individual nodes and not the sum of their parts. Currently, transactions on the EVM are not parallelizable, and every transaction is executed in sequence globally. The scalability problem then has to do with the idea that a blockchain can have at most 2 of these 3 properties:

- Decentralization
- Scalabiltity
- Security

If we have scalability and security, it would mean that our blockchain is centralized and that would allow it to have a faster throughput. Right not, Ethereum is decentralized and secure.

How can we break this trilemma to include scalability in the current model? Well can’t we just increase the block size, or in Ethereum’s case, the GAS_LIMIT, to increase throughput? While in theory this can be a right approach, the more we keep increasing it, the more mining will be centralized around nodes running on supercomputers that would bring a higher barrier to entry into the system.

A smarter approach is the idea of blockchain sharding, where we split the entire state of the network into a bunch of partitions called shards that contain their own independent piece of state and transaction history. In this system, certain nodes would process transactions only for certain shards, allowing the throughput of transactions processed in total across all shards to be much higher than having a single shard do all the work as the mainchain does now.

**Before we dive into how a sharded blockchain actually works, let’s go over some important vocabulary:**

- State: the entire set of information that describes a system at any point in time. In Ethereum, this is the current account set containing current balances, smart contract code, and nonces at a given moment. Each transaction alters this state into an entirely new state.
- Transaction: an operation issued by a user that changes the state of a system
- Merkle Tree: a data structure that can store a large amount of data via cryptographic hashes. Merkle trees make it easy to check whether a piece of data is part of the structure in a very short amount of time and computational effort.
- Receipt: a side-effect of a transaction that is not stored in the state of the system, but is kept in a Merkle tree so that its existence can be easily verified to a node. Smart contracts logs in Ethereum are kept as receipts in Merkle Trees, for example.
 
![image](https://i.imgur.com/MGE8o59.png)

With this in mind, let’s take a look at the structure of a sharded system. First of all, we would have nodes called collators on a certain shard that would be tasked with creating a collation, which is a specific structure that encompasses important information about the shard in question.

These collations are like mini-descriptions of the state and the transactions on a certain shard. They each have something a collation header, which is a piece of data containing

- Information about what shard the collation corresponds to (let’s say shard 10)
- Information about the current state of the shard before all transactions are applied
- Information about what the state of the shard will be after all transactions are applied
- Digital signatures from at least 2/3 of all collators on the shard affirming a collation is legit

In this new blockchain, a block is valid when

- transactions in all collations are valid
- the state of the collations is the same as the current state of the collations before the transactions
- the state of collations after the transactions is the same as as what the collation headers specified
- collations are signed by 2/3’s of all collators
- What about if a transaction happens across shards? For example, what if I send money from an address that is in shard 1 to an address in shard 10? One of the most important parts of this system is the ability to communicate across shards, otherwise we have accomplished nothing new. Here’s where the idea of receipts comes into play and how it can allow for the aforementioned scenario to work.

### Raul (Address on Shard 1) Wants to Send 100 ETH to Jim (Address on Shard 10)

1. A transaction is sent to Shard 1 that reduces Raul’s balance by 100 ETH and the system waits for the transaction to finalize
2. A receipt is then created for the transaction that is not stored in the state but in a Merkle root that can be easily verified
3. A transaction is sent to Shard 10 including the Merkle receipt as data. Shard 10 checks if this receipt has not been spent yet
4. Shard 10 processes the transaction and increases the balance of Jim by 100 ETH. It then also saves the fact that the receipt from Shard 1 has been spent
5. Shard 10 creates a new receipt that can then be used in subsequent transactions

### This Sounds Cool, But What Are Some Pitfalls?

The problems with sharded blockchains become more apparent once we consider that possible attacks on the network. A major problem is the idea of a Single-Shard Takeover Attack, where an attacker takes over the majority of collators in a single shard to create a malicious shard that can submit invalid collations. How do we solve this problem?

![image](https://i.imgur.com/VknJRZg.png)
Credits Hsiao-Wei Wang

The Ethereum Wiki’s [Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ) suggests random sampling of collators on each shard. The goal is so these validators will not know which shard they will get in advance. Every shard will get assigned a bunch of collators and the ones that will actually be validating transactions will be randomly sampled from that set.

Proof of stake makes this quite trivial because there is already a set of global validators that we can select collators from. The source of randomness needs to be common to ensure that this sampling is entirely compulsory and can’t be gamed by the validators in question.

Additionally, there are some potential latency problems with doing this sort of random sampling. Imagine you ran an Ethereum node and have already synced up with the entire blockchain history to begin doing transactions. What if after a few blocks you had to completely sync again with a new chain? This is what would happen with the reshuffling of validator nodes because they would each need to re-download a new shard when they are randomly assigned as collators, introducing a lot of potential overhead.

To read more about potential security risks and a detailed approach to this and other problems, check out the [Ethereum Sharding FAQ](https://github.com/ethereum/wiki/wiki/Sharding-FAQ).

### This Sounds So Complex for Solidity Devs and Ethereum Users to Understand! How Will We Educate Them on Sharding?

They don’t need to. Sharding will exist exclusively at the protocol layer and will not be exposed to developers. The Ethereum state system will continue to look as it currently does, but the the protocol will have a built-in system that creates shards, balances state across shards, gets rid of shards that are too small, and more. This will all be done behind the scenes, allowing devs to continue their current workflow on Ethereum.

### Beyond Scaling: Super-Quadratic Sharding and Incredible Speed Gains

To go above and beyond, it is possible that Ethereum will adopt a super-quadratic sharding scheme (which in simple English means a system built from shards of shards). Such complexity is hard to imagine at this point but the potential for scalability would be massive. Additionally, a super-quadratically-sharded blockchain will offer tremendous benefits to users, decreasing transaction fees to negligible quantities and serving as a more general purpose infrastructure for a variety of new applications.

### Resources and Where to Get Started

Ok so now you want to start coding a sharded blockchain! How do you begin? At the most basic level, the proposed initial implementation will not work through a hard fork, but rather through a smart contract known as the Validator Manager Contract that will take control of the sharding system.

![image](https://i.imgur.com/KOyfwZ9.png)

This VMC will manage shards and the sampling of proposed collators from a global validator set and will take responsibility for the global reconciliation of all shard states. Vitalik has outlined a fantastic reference doc for implementing sharding here: https://github.com/ethereum/sharding/blob/develop/docs/doc.md

To get explore this VMC architecture in detail and to learn more about how the system works, check out the following resources:

- Sharding FAQ: https://github.com/ethereum/wiki/wiki/Sharding-FAQ
- Ethereum Sharding Technical Overview: https://medium.com/@icebearhww/ethereum-sharding-and-finality-65248951f649
- Sharding Reference Doc: https://github.com/ethereum/sharding/blob/develop/docs/doc.md

### Wanna Join My Team?

Are you familiar with the inner workings of the Ethereum protocol? Are you a golang developer with experience using the Geth Ethereum client? Do you want to work with me and a team of talented developers building the first sharding client for Geth? Feel free to reach out to me raul@prysmaticlabs.com and let’s get to work!
