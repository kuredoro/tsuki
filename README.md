![tsuki-banner](https://i.imgur.com/4YaQrQS.png)

tsuki
[![codecov](https://codecov.io/gh/kureduro/tsuki/branch/master/graph/badge.svg)](https://codecov.io/gh/kureduro/tsuki)
==============

A simple distributed file system (with SPoF).

## Contribution

* **Artem Bakhanov**: nameserver creation, writing report, debugging, Docker image creation;
* **Grigoriy Dolgov**: fileserver creation, writing report, testing;
* **Mohamad Ziad AlKabakibi**: client creation, writing report.

Protocols are designed by Artem Bakhanov and Grigoriy Dolgov.

## How to run

### Locally

To run a name server or a fileserver locally, one could run
```
$ go run cmd/tsukinsd/*
```
or
```
$ go run cmd/tsukifsd/*
```
respectively.

The client provides a friendly CLI interface (help messages included). To build and run it, run
```
$ go build -o tsuki cmd/tsuki/*
$ ./tsuki
```

### In a container

The Docker Hub repository can be found [here](https://hub.docker.com/r/artembakhanov/project2).

To run this code one needs first to get a virtual network for safety and security so that the servers can communicate. That can be either VPS provided network or Docker's overlay network. 
1. All the fileservers must be safely run (docker-compose example is below). The fileservers should have static IP addresses and open ports (our recommendation: 7000 for client-storage communication; 7001 should be hidden from the public internet).
2. Then one name server should be run on a different host (or VM) and with different ports (we use 7070 for client-nameserver communication and 7071 for nameserver-fileserver one). Before running the nameserver one must specify the parameters of the DFS they want in the `config.poml` file. For instance, to specify the number of replicas to 3 they must write `replicas=3`. The config file is provided by default.
3. The DFS will start working so that any client can invoke `init` procedure to start working with the system.

Now let us talk about running more specifically.
To run the name server (after negotiating port and address issues) one needs to create a docker-compose file as follows:
```dockerfile=1
version: 3

services:
  nameserver:
    image: artembakhanov/project2:tsukinsd
    ports:
    # other can be specified
      - 7070:7070
      - 7071:7071
    volumes:
      - ./data:.
```

Run `docker-compose up` on the nameserver host and it will start working if everything is specified correctly.

```dockerfile=1
version: 3

services:
  fileserver:
    image: artembakhanov/project2:tsukifsd
    # other can be specified
    ports:
      - 7000:7000
      - 7001:7001
    volumes:
      - ./data:.
```

## Architecture
The architecture is very similar to the popular ones. There is one server (node) called "nameserver" (NS) with which any client contacts in the first place. This server decided whether to give the client an approval to upload its file or download a requested one. On the diagram below you can see how the channels are working. Other servers are fileservers (FS) and are not aware of what is going on in the system; their primary goal is to be alive all the time (if possible) and store chunks of data. They can get a command from the nameserver and must obey it and complete it as soon as possible. They also send periodic messages called heartbeats to the nameserver with the intention that in case of any failure the nameserver will know who failed and will take necessary actions. 

![](https://i.imgur.com/vkaX1SF.png)

The diagram above shows all the possible channels of communication in client-NS-FS structure. Note that all the messages between NS and FS are hidden from the public network.

### Nameserver
Let us now go to the nameserver architecture. The main goal of the nameserver is to contain all the information about the current state of the file system. 
#### Filetree
The file system is logical and stored in the RAM as a hierarchical structure (a tree). The example is provided below:
![](https://i.imgur.com/XOjp1AR.png)

#### Tree node
In the tree, there can be two types of nodes: a file, which cannot have children, and the tree, which has children but no data (no chunks). In our code, the tree is organized as a hashmap from the full path (we call it path address) to the node itself. Each node has references to all its children and to its parent so knowing the address of one node we can traverse in tree easily.

Let us look at the implementation of the node itself.
![](https://i.imgur.com/r07zeYo.png)
Each tree node knows the general information about the file: its address, size, creation date, etc. Also, it has a small piece of information provided by the tree: its children and its parent. It also should contain information about whether it was removed or not, since we use lazy removing: just mark some node as removed and remove it from the hashmap but the node itself will stay in the tree structure and will be removed eventually. It is a very nice solution in case of some expensive operations like directory removing: just mark one directory as dead and return success to the client and only then it will work with the mess it created.

The chunk structure is very simple.
![](https://i.imgur.com/sR4sWgl.png)
Chunk always has a unique id. It is impossible that two chunks have the same ID so the UUID is used for their identification. It also contains the information to which file it belongs. Other information (in red) is about the status of the chunk. The general status may be **PENDING** (the file is just created but the client has not uploaded this chunk yet), **OK** (the client can download the file), **DOWN** (no server can provide this chunk; the file is **DEAD**) and **OBSOLETE** (a file to which this chunk belongs is removed). One of the tasks of the NS is to keep the number of OK replicas to be the same as the configuration number of replicas.

#### Public communication service
Provides simple REST API service to the clients (`/upload`, `/touch`, `/rmfile`, etc)

#### Private communication service and heartbeat manager
Provides REST API service to fileservers.

Also, there is a simple heartbeat message that has 2 different timeouts: soft (11 secs) and hard (61 secs). The first decides if the FS will be given to the client and the last when the nameserver should start the relocation of the chunks to maintain replicas. Each fileserver should send one heartbeat message in 3 seconds.

### Fileserver
The fileserver architecture is a bit simpler than the one of the nameserver. The fileserver goal is to store the chunks of data and to obey all the nameserver commands.

All the chunks are stored in a flat directory since hierarchy is maintained on the nameserver. 
Another service maintains chunk and token states. 

## Communication protocols

### Key highlights
* Support for concurrent reads and writes
* Support for multiplexed downloads and uploads
* Load balancing
* Client authentication
* Support for different replica counts
* Rejecting writes in case of memory deficiency [TODO]

### Overview

The communication protocols are built on top of HTTP/1.1 and represent a REST API. The full description of the API can be found on the repository's wiki.

The main motivations for the design of these protocols are security and scalability.

An **authentication system** was incorporated into the protocols and represents a **token-based authentication**. It allows controlling what actions and on what chunks may be performed by clients on an individual basis.

Another interesting decision is that **chunks** are **immutable**. The removal of chunks, a **purge request**, respects users of the chunks. Before the actual deletion of the data, it **waits until** all **tokens** associated with them **expire**. The **chunks** to be purged are **marked** *obsolete* and **token emission** for them is **halted**. This scheme permits safe removal of files in case of concurrent access by multiple clients.

Using the removal primitive, the updates on files may be implemented in three phases: purge chunks, upload new ones and replace chunks associated with the file in the nameserver's database.

There is no separate interface for the replication process between fileservers. To replicate a chunk, FS sends it to the client-port of the destination FS, effectively **reusing the logic written for the client**. And prior to this, destination FS receives an expect request for that particular chunk from the nameserver. The **orchestration** is fully contained within the nameserver. It produces a sequence of messages, addressed to different fileservers, waits for confirmations of replicas, and decides what to do next. The replication process is sped up by utilizing **epidemic propagation**.

For the case of **slow network** channels on DFS' side, the client is able to **download** and **upload** chunks from and to **multiple** servers **simultaneously**. The number of servers is generally the number of replicas (if there are enough servers, of course). If clients don't utilize multiplex data loading, the servers to be requested are selected in Round-Robin fashion, which represents a load balancing mechanism.

The nameserver populates the **pool of trusted servers, PTS,** by probing fileservers. Until probed, fileservers can be accessed and modified by anyone. Authentication is disabled. Once probed, fileservers recognize the leader, permanently remember it, and block any further data-sensitive requests coming from unknown addresses.

Finally, we've introduced a notion of **soft timeouts** on heartbeats from the fileservers. Upon reaching this time of inactivity from a fileserver, the nameserver marks it as potentially dead and prevents its address to appear in responses to clients' requests for data. This improves the chances that the clients would be able to download and upload data reliably.

### What can be improved

* **Protection against write-write concurrency**
    The update method implementation described here is not consistent, when multiple writes are concurrent.

* **Replication canceling (in case of immediate overwrite)**
  If the above problem is solved and the order of writes is consistent, then an optimization can be employed: cancel replication and purge chunks of the outdate copy.
  
* **High network consumption**
  The replication is done on chunk-by-chunk basis: chunks are replicated one by one as they are being downloaded. This negatively affects network performance. A possible solution is to use the buffering of requests for replication at the FS side. Or, alternatively, employ *batch replication requests* that will ask to replicate multiple chunks simultaneously to a single target (already implemented in FS, but not used).
  
* **Authentication engine: users, passwords, permissions**
  Because authentication permeates through the protocol design, our code needed to support general cases of this idea. This made code flexible enough to allow relatively easy implementation of "users", "ownership", and "permissions" on files, much as in the physical file systems.

* **Data compression**
  Data may be compressed via DEFLATE or any other relatively fast compression algorithm to save network bandwidth.
  
* **Stateful name server**
  In the current design, it is assumed that the nameserver never fails. But this is exactly the area that can be improved! Logging and snapshotting of NS' state, and consequent resurrection from them after failure is a possible solution to this problem.

* **Data-preserving failures**
  For simplicity, we've assumed that if FS fails, all the data it stored is also lost. Obviously, if FS' host was just restarted, no data was lost and NS should be fully aware of that fact and utilize the chunks that survived efficiently.
  
* **Timeouts and requests for new fileservers**
  If FS fails in the middle of data transfer, the subject should be able to request from NS a new server to restart or continue (for write and read, respectively) the transaction.
