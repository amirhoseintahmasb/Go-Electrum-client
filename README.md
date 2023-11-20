# Go-Electrum-Client

Golang Electrum Client.

## Electrum client wallet in Go

- Electrum client multi-coin wallet library 
- ElectrumX chain server interface


## Introduction
Go-Electrum-Client is a Go-based client for interacting with Bitcoin Electrum servers. It provides functionalities for connecting to the Bitcoin network, managing wallets, and performing transactions.

## Features
- Connect to Bitcoin Electrum servers on various networks (mainnet, testnet, regtest, simnet).
- Manage Bitcoin wallet operations.
- Synchronize blockchain headers.
- Send and receive Bitcoin transactions.

## Getting Started

### Prerequisites
- Go (version 1.20 or newer)
- Access to a Bitcoin Electrum server

### Installation
Clone the repository and navigate into the project directory:
```
git clone 
cd go-electrum-client
```

### Configuration
Edit the configuration settings in `btcConnect.go` to specify the network (mainnet, testnet, etc.) and the Electrum server details.

### Running the Client
Run the client using:
```
go run btcConnect.go
```

## Usage
After running the client, you can perform operations such as:
- Generating a new Bitcoin address.
- Checking your wallet balance.
- Sending and receiving Bitcoin.

## Contributing
Contributions to the Go-Electrum-Client project are welcome. Please ensure that your contributions adhere to the project's coding standards and submit a pull request for review.

## License
This project is licensed under the [LICENSE] - see the LICENSE file for details.

