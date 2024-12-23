#### Proxy for SolarmanV5 data-loggers in TCP-Client mode

---

* Multi logger, multi client
  *  one logger can communicate with multiple clients, no client connections limit
     (limit is 5 when the logger is in TCP-Server mode)
  *  all clients communicate with the respective datalogger (serial number based)  

* Packets routing based on serial number
* Single point of communication with all data-loggers - one ip, one port (8899)
* Only V5 encoded packets are processed
* Recommended client [pysolarmanV5](https://github.com/jmccrohan/pysolarmanv5.git)

---
##### Usage
* Proxy start
    ```console
    go-solarmanV5-proxy <address> <port for data-logger connections>
    ```
    e.g.
    ```console
    go-solarmanV5-proxy 192.168.1.3 12345
    ```
 * Options
   * `-debug` flag can be used to see what's going on under the hood :sunglasses:
   * `-silent` flag will make the proxy completely silent
   * `-bcast` activates a broadcast listener/server
   * `-buffered` will activate sequential communication with the datalogger 
 * all messages are logged to stdout for now 
* Data logger configuration (config_hide.html)
![image](img/logger_tcp_srv.png "Config")

All clients then can be connected to port 8899 of the proxy server

When the `-bcast` flag is used the proxy will respond to logger scan requests. All dataloggers currently connected will be listed.

The `-buffered` flag allows much more stable communication with the inverter when 2 or more clients are used.

---
#### Build

```console
make 
```
* The binary will be placed in the `build` dir 

----
#### Install
```console
go install github.com/githubDante/go-solarman-proxy@latest
```

* for ready to use binaries see [Releases](https://github.com/githubDante/go-solarman-proxy/releases) 