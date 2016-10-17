package main

import "net"
import "fmt"
import "bufio"
import "./myUtils"
import "time"
import "strings"
import "strconv"
import "net/rpc"
//import "reflect"

//CONSTANTS
const SERVER_IP string = "";
const SERVER_PORT string = "8080";
const NOT_IN_ROOM_ERR string = "You are not in a room yet";
const NO_ROOM_NAME_GIVEN_ERR string = "You must specify a room name";
const ROOM_NAME_NOT_UNIQUE_ERR string = "The room name you have specified is already in use";
const CLIENT_LEFT_ROOM_MESSAGE string = "CLIENT HAS LEFT THE ROOM";
const CLIENT_JOINED_ROOM_MESSAGE string = "CLIENT HAS JOINED THE ROOM";
const MAX_CLIENTS int = 10;
const DAY_DURATION time.Duration = 24*time.Hour;
const ROOM_DURATION_DAYS time.Duration = 7*DAY_DURATION;
const TIMEOUT_DURATION time.Duration = 2*time.Minute;


//COMMANDS
const COMMAND_PREFIX string = "/";
const HELP_COMMAND string = COMMAND_PREFIX+"help";
const QUIT_COMMAND string = COMMAND_PREFIX+"quit";
const CREATE_ROOM_COMMAND string = COMMAND_PREFIX+"createRoom"; //creates a room with the name of the first argument given
const LIST_ROOMS_COMMAND string = COMMAND_PREFIX+"listRooms"
const JOIN_ROOM_COMMAND string = COMMAND_PREFIX+"join";//   /join roomname will add a user to a rooms list of clients and switch the user to that room
const CURR_ROOM_COMMAND string = COMMAND_PREFIX+"currentRoom";
const CURR_ROOM_USERS_COMMAND string = COMMAND_PREFIX+"currentUsers";
const LEAVE_ROOM_COMMAND string = COMMAND_PREFIX+"leaveRoom";

var HELP_INFO = [...]string {"help and command info:",
 HELP_COMMAND+": use this command to get some help",
 QUIT_COMMAND+": Safely exit the system",
 CREATE_ROOM_COMMAND+" roomName : creates a room with the name roomName",
 LIST_ROOMS_COMMAND+": lists all rooms available for joining",
 JOIN_ROOM_COMMAND+" roomName: adds you to a chatroom",
 CURR_ROOM_COMMAND+": tells you what your current room is",
 CURR_ROOM_USERS_COMMAND+": gives a you a list of users in a room",
 LEAVE_ROOM_COMMAND+" removes you from current room",
}
var ClientArray []Client;
var RoomArray []*Room;
//STRUCTURES
/*****************Rooms*****************/
type Room struct{
  name string;
  clientList []*Client;
  createdDate time.Time;
  lastUsedDate time.Time;//This date is updated when clients leave the room, a room will be deleted if it hasnt been accessed in 7 days AND its empty
  chatLog []*ChatMessage;
  creator *Client;
}

//Creates a new room, with a specified roomCreator and roomName. the room will be added to the global list of rooms, if room is not unique, the client will be messaged
func createRoom(roomName string, roomCreator *Client) *Room {
  //check uniqueness of name, warn user and abort if not unique
  if isRoomNameUnique(roomName) == false {
    roomCreator.messageClientFromServer(ROOM_NAME_NOT_UNIQUE_ERR)
    return nil
  }
  var newRoom = Room{
    name: roomName,
    clientList: make([]*Client, 0),//room will start empty, we wont add the creator in
    createdDate: time.Now(),
    lastUsedDate: time.Now(),
    chatLog: nil,
    creator: roomCreator,
  }
  RoomArray = append(RoomArray, &newRoom);
  return &newRoom;
}

//checks the room name against the current list of rooms to make sure it is unique, returns true if it is, false if not
func isRoomNameUnique(roomName string) bool{
  for _, room := range RoomArray {
    if roomName == room.name{
      return false
    }
  }
  return true
}
//returns true if a user is already in the room, false otherwise
func (room Room) isClientInRoom(client *Client) bool {
  for _, roomClient := range room.clientList {
    if client.name == roomClient.name {
      return true;
    }
  }
  return false;
}
/***************************************/

/*****************MESSAGES*****************/

//Structure holding messages sent to a chat, stores meta information on the client who sent it
type ChatMessage struct {
  client *Client;
  message string;
  createdDate time.Time;
}

//creates a new instance of a ChatMessage and returns it
func createChatMessage(cli *Client, mess string) *ChatMessage {
 var chatMessage = ChatMessage{
   client: cli,
   message: mess,
   createdDate: time.Now(),
 }
 return &chatMessage;
}
/******************************************/

/*****************CLIENTS*****************/
//Clients have names, and a reader and writer as well as a link to their connection
//Client names are garenteed by the generateName fucntion to be unique for the duratoin of program execution (NOT persisted)
type Client struct
{
  currentRoom *Room;
  outputChannel chan string;
  name string;
}

func getClientByName(clientName string) *Client{
  for _, cli := range ClientArray{
    if cli.name == clientName{
      return &cli;
    }
  }
  return nil;
}
/*
creates a new client with a random name and returns the name
*/
func addClient() string{
   createOutputChannel := make(chan string);
   createName := myUtils.GenerateName();

    var cli  = Client{
    currentRoom: nil, //starts as nil because the user is not initally in a room
    outputChannel: createOutputChannel,
    name: createName,
  }
  ClientArray = append(ClientArray, cli);
  return cli.name
}


//adds message to the clients output channel, messages should be single line, NON delimited strings, that is the message should not include a new line
//the name of the sender will be added to the message to form a final message in the form of "sender says: message\n"
func (cli Client) messageClientFromClient(message string, sender *Client){
  message = string(sender.name)+" says: "+message+"\n";
  cli.outputChannel <- message;
}

//without a client argument assumes the message is coming from the server
func (cli Client) messageClientFromServer(message string){
  message = "Server says: "+message+"\n";
  cli.outputChannel <- message;
}
/**********************************/

//sends a message to the clients current room, this function will replacee the WriteToAllChans function which sends a message to every client on the server
func sendMessageToCurrentRoom(sender *Client, message string){
//check if the client is currently in a room warn otherwise
if sender.currentRoom == nil {
  //sender is not in room yet warn and exit
  sender.messageClientFromServer(NOT_IN_ROOM_ERR);
  return;
}
//get the current room and its list of clients
//send the message to everyone in the room list that is CURRENTLY in the room
room := sender.currentRoom;
chatMessage := createChatMessage(sender, message);
fmt.Println("current room UserArray: ")
fmt.Println(room.clientList)
for _, roomUser := range room.clientList {
  fmt.Println("looping room array user is: "+roomUser.name)
  //check to see if the user is currently active in the room
  if ((roomUser.currentRoom.name == room.name)) {
    roomUser.messageClientFromClient(chatMessage.message, chatMessage.client)
  }
}
//save the message into the array of the rooms messages
room.chatLog = append(room.chatLog, chatMessage);
}

/*
Checks if the line sent from the user includes a command
Commands will be in the form of /Command arg
this function will first check if the FIRST character of the clients string is a /,
if it is then it will attempt to parse and execute the command.
*/
func checkForCommand(message string, client *Client) {
  message = strings.TrimSpace(message);//strips the newlines from the string
  isCommand := strings.HasPrefix(message, COMMAND_PREFIX);//checks to see if the line starts with /
  if(isCommand){
    //parse command line, commands should be in the exact form of "/command arg arg arg" where args are not required
    parsedCommand := strings.Split(message, " ")
    if parsedCommand[0] == HELP_COMMAND {

    } else if parsedCommand[0] == QUIT_COMMAND {
      processQuitCommand(client);
    } else if parsedCommand[0] == CREATE_ROOM_COMMAND {
      // not enough arguments to the command
      if len(parsedCommand) < 2{
        client.messageClientFromServer(NO_ROOM_NAME_GIVEN_ERR)
      }else{
        processCreateRoomCommand(client, parsedCommand[1]);
      }
    } else if parsedCommand[0] == LIST_ROOMS_COMMAND {
      processListRoomsCommand(client);
    } else if parsedCommand[0] == JOIN_ROOM_COMMAND {
      //not enough given to the command
      if len(parsedCommand) < 2{
        client.messageClientFromServer(NO_ROOM_NAME_GIVEN_ERR)
      }else{
        processJoinRoomCommand(client, parsedCommand[1]);
      }
    } else if parsedCommand[0] == CURR_ROOM_COMMAND {
      processCurrRoomCommand(client);
    }else if parsedCommand[0] == CURR_ROOM_USERS_COMMAND{
      processCurrRoomUsersCommand(client);
    }else if parsedCommand[0] == LEAVE_ROOM_COMMAND{
      processLeaveRoomCommand(client)
    }

  } else { // message is not a command
    sendMessageToCurrentRoom(client, message);
  }
}


//*****************RPC SERVER OBJECT************************
//here we will create a server object and expose the processing commands to the client, this way the client will be able to
//directly call the below functions

type Server int;

type CreateRoomArgs struct{
  ClientName string;
  RoomName string;
}


func (t *Server) ConnectMe(name string, reply *string) error {
  *reply = addClient();
  return nil; //TODO errors
}

func (server *Server) MessageClient(clientName string, reply *string) error {
cli := getClientByName(clientName);
*reply = <-cli.outputChannel;
return nil;
}

//creates a room and logs to the console
func (server *Server) ServerProcessCreateRoomCommand(c *CreateRoomArgs, reply *string) error {
  client := getClientByName(c.ClientName);
  roomName := c.RoomName;
  room := createRoom(roomName, client);
  if room == nil { //name of room was not unique
    return nil
  }
  client.messageClientFromServer(client.name+" created a new room called "+roomName)
  return nil
}

func processLeaveRoomCommand(client *Client){
  removeClientFromCurrentRoom(client);
  client.messageClientFromServer("You have left the room.")
}

//sends a list of the current users in the room to the client
func processCurrRoomUsersCommand(client *Client){
  //check if the user is in a room
  if client.currentRoom == nil{
    client.messageClientFromServer(NOT_IN_ROOM_ERR)
    return
  }
  client.messageClientFromServer("Current users in "+client.currentRoom.name+" are:")
  for _, users:= range client.currentRoom.clientList {
    client.messageClientFromServer(users.name);
  }
}


//sends a message to the client telling them which room they are currently in, if not in a room, inform the user
 func processCurrRoomCommand (client *Client){
   if client.currentRoom == nil{
     client.messageClientFromServer(NOT_IN_ROOM_ERR)
     return
   }
   client.messageClientFromServer("current room: "+client.currentRoom.name);
 }

//Loops through the HELP_INFO array and sends all the lines of help info to the user
func (server *Server)ProcessHelpCommand(clientName string, reply *string) error{
       client := getClientByName(clientName);
       for _, helpLine := range HELP_INFO{
         client.messageClientFromServer(helpLine);
       }
       //TODO error
       return nil
}

//quits the client from the server
func processQuitCommand(client *Client){
  //client.messageClientFromServer("Goodbye");
  removeClientFromCurrentRoom(client);
  removeClientFromSystem(client);
}

func processTimeout(client *Client){
  defer processQuitCommand(client)
  client.messageClientFromServer("TIMEOUT")
  time.Sleep(2*time.Second)
}

//This function will remove the client from the Client Array, this function is intended to be used as part of the processQuitCommand
func removeClientFromSystem(client *Client){
  //finds the client and removes them from the ClientArray
  for i,systemClients := range ClientArray{
    if client.name == systemClients.name {
      ClientArray = append(ClientArray[:i], ClientArray[i+1:]...)//deletes the element
    }
  }
  fmt.Println("there are currently: "+strconv.Itoa(len(ClientArray))+" clients connected");
}

//creates a room and logs to the console
func processCreateRoomCommand(client *Client, roomName string){
  room := createRoom(roomName, client);
  if room == nil { //name of room was not unique
    return
  }
  message := room.creator.name+" created a room called: "+room.name
  fmt.Println(message)
  client.messageClientFromServer(message)
}

//sends the list of rooms to the client
func processListRoomsCommand(client *Client){
  client.messageClientFromServer("List of rooms:")
  for _, roomName := range RoomArray{
    client.messageClientFromServer(roomName.name);
  }
  client.messageClientFromServer("");
}

//returns true of the room was joined successfully, returns false if there was a problem like the room does not exist
func processJoinRoomCommand(client *Client, roomName string) bool{
  //start by checking if the room exists
  roomToJoin := getRoomByName(roomName);
  if roomToJoin == nil{ //the room doesnt exist
    fmt.Println(client.name+" tried to enter room: "+roomName+" which does not exist");
    client.messageClientFromServer("The room "+roomName+" does not exist")
    return false;
  }
  //Room exists so now we can join it.
  //check if user is already in the room
  //add user to room if not in it already
  if roomToJoin.isClientInRoom(client) {
      //all good
  } else {
    removeClientFromCurrentRoom(client);
    roomToJoin.clientList = append(roomToJoin.clientList, client);// add client to the rooms list
  }
  //switch users current room to room
  client.currentRoom = roomToJoin;
  fmt.Println(client.name+" has joined room: "+client.currentRoom.name)
  sendMessageToCurrentRoom(client, CLIENT_JOINED_ROOM_MESSAGE)
  //display all messages in the room
  displayRoomsMessages(client, roomToJoin)
  //
  return true
}

func removeClientFromCurrentRoom(cli *Client){
//not in a current room so just return
  if cli.currentRoom == nil {
    return;
  } else {
    sendMessageToCurrentRoom(cli, CLIENT_LEFT_ROOM_MESSAGE)
    cl := cli.currentRoom.clientList;
    for i,roomClients := range cl{
      if cli == roomClients {
        cli.currentRoom.clientList = append(cl[:i], cl[i+1:]...)//deletes the element
        cli.currentRoom.lastUsedDate = time.Now();
      }
    }
    cli.currentRoom = nil;
    return
  }

}
//diplays to the user all the messages of the chatroom, intended to be used when a user first joins a room
func displayRoomsMessages(client *Client, room *Room){
  //loop through the chatlog and send the user everything
  //just so the user doesnt get an empty message
  if room.chatLog == nil{
    return
  }
  client.messageClientFromServer("-----Previous Log-----")
  for _, messages := range room.chatLog {
    client.messageClientFromClient(messages.message, messages.client)
  }
  client.messageClientFromServer("----------------------")

}
//checks to see if a room with the given name exists in the RoomArray, if it does return it, if not return nil
func getRoomByName(roomName string) *Room{
  for _, room := range RoomArray{
    if room.name == roomName{
      return room;
    }
  }
  return nil;
}

//sends a message to the client connection "SERVER FULL" and then closes the connection
func sendServerIsFullMessage(conn net.Conn){
  writer := bufio.NewWriter(conn);

  //send FULL Message to Client
  _, error := writer.WriteString("SERVER FULL")
  if error != nil{
    fmt.Println(error)
  }
  //flushing is necessary, the writeString only takes in the string, the flush function pushes it out to the user
  flushError := writer.Flush()
  if flushError != nil {
    fmt.Println(flushError)
  }

  conn.Close();
}

//intended to be run continously on a thread, this function will look at the usage of rooms and if the room hasent been used for 7 days,
// it will be closed. If a room has no active users and the last user left over 7 days ago the room will be closed. this function will check the room
//status every minute
func manageRooms(){
  for{ //loop forever
    for i, rooms := range RoomArray{
      //for each room in the array we need to check if its been used, if not, remove it
      sinceLastUsed := time.Since(rooms.lastUsedDate)
      if len(rooms.clientList) == 0 && sinceLastUsed > ROOM_DURATION_DAYS{ //room is empty and time since use is longer than allowed duration
        RoomArray = append(RoomArray[:i], RoomArray[i+1:]...)//deletes the element
      }
      //else don't do anything
    }
    time.Sleep(time.Minute)//sleep the loop to lower processing
  }
}



//Main function for starting the server, will open the server on the SERVER_IP and the SERVER_PORT
func main() {
  fmt.Println("Launching server...")
  //Start the server on the constant IP and port
  ln, connectError := net.Listen("tcp", ":"+SERVER_PORT)
  //check for errors in the server starup
  if connectError != nil {
    fmt.Println("Error Launching server "+ connectError.Error())
  }else{
    fmt.Println("Server Started")
  }
  //start RPC
  server := new(Server);
  go manageRooms();//start the room manager
  rpc.Register(server);
  rpc.Accept(ln)
  fmt.Println("here")

  // run loop forever, accept connections when they come and add them to the connection array and then call the addClient function one
  for {
    conn, _ := ln.Accept();
    if len(ClientArray) < MAX_CLIENTS{//server can have more clients
    }else{
      sendServerIsFullMessage(conn)
    }
  }
}
