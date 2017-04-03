/* Copyright 2017, Ashish Thakwani. 
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.LICENSE file.
 */
package forcecmd

import (
    "fmt"
    "encoding/json"
    "os"
    "regexp"
    "strconv"
    "net"
    "log"
    "runtime"

    "../config"
    netutil "github.com/shirou/gopsutil/net"
    ps "github.com/shirou/gopsutil/process"
)


func blockForever() {
    select { }
}

/*
 *  Get Parent process's pid and commandline.
 */
func getProcParam(int32 p) (*ps.Process, cmd string) {

    // Init process struct based on PID
    proc, err := ps.NewProcess(p)
    utils.Check(err)

    // Get command line of parent process.
    cmd, err := proc.Cmdline()
    utils.Check(err)
    
    return pid, cmd
}

/*
 *  Get tunnel connections parameters in host struct
 */
func getConnParams(pid int32, h *Host) {
    // Get SSH reverse tunnel connection information.
    // 3 sockers are opened by ssh:
    // 1. Connection from client to server
    // 2. Listening socket for IPv4
    // 3. Listening socket for IPv6
    conns, err := netutil.ConnectionsPid("inet", pid)
    utils.Check(err)
    log.Println(conns)

    for _, c := range conns {
        // Family = 2 indicates IPv4 socket. Store Listen Port
        // in host structure.
        if c.Family == 2 && c.Status == "LISTEN" {
            h.ListenPort = int32(conn.Laddr.Port)
        }

        // Store Established connection IP & Port in host structure.
        if c.Family == 2 && c.Status == "ESTABLISHED" {
            h.RemoteIP   = c.Raddr.IP
            h.RemotePort = c.Raddr.Port
        }
    }
}


/*
 *  Get Client configuration parameters in host struct
 */
func getConfigParams(h *Host) {
    
    // Get Client config which should be the last argument
    cfgstr := os.Args[len(os.Args) - 1]
    log.Println(cfgstr)
    
    // Conver config to json
    cfg := Config{}
    json.Unmarshal([]byte(cfgstr), &cfg)
    
    // Update and log host var
    h.ServicePort = cfg.Port                
    h.Config = cfg
    h.Uid = os.Getuid()
    
}

/*
 *  Match string with regex.
 */
func match(regex string, str string) bool {

    if len(str) > 0 {
        // find server with current user ID using command line match 
        ok, err := regexp.MatchString(pstr, cmd)
        utils.Check(err)

        // If found send host var to server
        if ok {
            return true
        }
    }
    
    return false
}

/*
 *  Match string with regex.
 */
func writeHost(pid int32, h *Host) {

    // Form Unix socket based on pid 
    f := config.RUNPATH + strconv.Itoa(int(pid)) + ".sock"
    log.Println("SOCK: ", f)
    c, err := net.Dial("unix", f)
    utils.Check(err)
    
    defer c.Close()

    // Convert host var to json and send to server
    payload, err := json.Marshal(h)
    utils.Check(err)
    
    // Send to server over unix socket.
    _, err = c.Write(payload)
    utils.Check(err)
}

/*
 *  Get connection information of ssh tunnel and send the 
 *  information to server.
 */
func SendConfig() {
    
    // Get parent proc ID which will be flock's pid.
    ppid := os.Getppid()
    log.Println("ppid = ", ppid)
    
    // Get parent process params
    pproc, pcmd := getProcParam(int32(ppid))
    log.Println("Parent Process cmdline = ", pcmd)

    // Get SSH process ID
    spid, err := pproc.Ppid()
    utils.Check(err)

    // Get SSH proc and command line, 
    sproc, scmd := getProcParam(spid)
    log.Println("SSH Process cmdline = ", scmd)

    //Host to store connection information
    var h Host
    h.Pid = int32(ppid)
    
    //Get socket connection parameters in host struct
    getConnParams(spid, &h)
    
    //Get client config parameters in host struct
    getConfigParams(&h)

    //Log complete host struct.
    log.Println(h)
    
    // Scan through all system processes to find server
    // based on current user id.
    // This is done by regex matching the UID with 
    // commandline of server
    pids, _ := ps.Pids()
    for _, p := range pids  {
        
        // Get proc and commandline based on pid
        pr, cmd := getProcParam(p)

        // Check if server 
        ok := match(fmt.Sprintf(`trafficrouter .* -uid %d .*`, os.Getuid()), cmd)
        // If found send host var to server
        if ok {    
            log.Printf("Found Server Process %s, pid = %d\n", cmd, p)
            writeHost(p, h)
        }            
    }
    
    blockForever()
}
