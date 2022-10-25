# tape



![tape](./tape.jpeg)



## Components

| Component   | Amount                                        |
| ----------- | --------------------------------------------- |
| Signer      | config option `signerNum`                       |
| Proposer    | config option `connNum` × endorser number |
| Integrator  | config option `integratorNum`                       |
| Broadcaster | config option `broadcasterNum`               |
| Observer    | 1                                             |



## Workflow

Main workflow:

1. Initiator creates specified number (CLI option `--number`) of unsigned transaction proposals
2. Signers sign proposals
3. Proposers send proposals to and collect endorsement from endorsers
4. Integrators sign each endorsed proposal and assemble it into an envelope
5. Broadcastors send envelopes to orderer

In the meanwhile, an observer keeps listening to the blocks newly added to the channel and collect data from them.



## Usage

### Build
- local: `go build ./cmd/tape`       
- docker: `docker build -f Dockerfile -t tape .`

### Run

1. Link to crypto materials: `ln -sf $YOUR_PROJECT/organizations`
2. End-to-End Run   
    ```bash
    # create 2000 accounts
    ./tape -c config.yaml --txtype put --endorserGroupNum 1 --number 2000 --seed 2333 --rate 1000 --burst 50000 --broadcasterNum 5 --connNum 4 --clientPerConnNum 4 
    # start 2000 payment transactions
    ./tape -c config.yaml --txtype conflict --endorserGroupNum 1 --number 2000 --seed 2333 --rate 1000 --burst 50000 --broadcasterNum 5 --connNum 4 --clientPerConnNum 4 
    ```
3. Breakdown      
    ```bash
    rm ENDORSEMENT
    # phase1
    ./tape --no-e2e -c config.yaml --txtype put --endorserGroupNum 1 --number 2000 --seed 2333 --rate 1000 --burst 50000 --broadcasterNum 5 --connNum 4 --clientPerConnNum 4 
    # phase2
    ./tape --no-e2e -c config.yaml --txtype put --endorserGroupNum 1 --number 2000 --seed 2333 --rate 1000 --burst 50000 --broadcasterNum 5 --connNum 4 --clientPerConnNum 4 
    ```

### Result

Save output to file for analysis: `./tape -c config.yaml -n 10000  > log.transactions `

#### latency breakdown
```
cat log.transactions | python3 scripts/latency.py > latency.log  
```

**Output format:** txid [#1, #2, #2]

1. endorseement: clients sends proposal => client receives enough endorsement
2. local_process: clients generate signed transaction based on endorsements
3. consensus & commit: clients send signed transaction => clients receive response (including consensus, validation, and commit)


#### conflict rate
```bash
bash scripts/conflict.sh
```



## TODO

1. zipfan distribution workload
2. add prometheus
