# irc

irc implements the IRC protocol to facilitate communication between clients via a server over TCP. This project contains a server program and a client program which are intended to be used together. Future versions may allow communication with other IRC servers or clients.

# dependencies

- This project is compiled with go1.7. You can get it [here](https://golang.org/dl/)
- The database runs on MySQL 5.7. Download it from [here](https://dev.mysql.com/downloads/mysql/)

# installation

To build this project, make sure you have Go installed, then in a console enter:

`go install -v all`

# server

To host a server on your localhost, first build the project. Then, navigate to $GOPATH/bin and run server.exe.

# client

To host a client on your localhost, first build the project. Then, navigate to $GOPATH/bin and run client.exe. Make sure a server program is running prior to starting any clients.
