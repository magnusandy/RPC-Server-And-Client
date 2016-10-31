package main

import "net"
import "fmt"
import "./myUtils"
import "time"
import "strconv"
import "net/rpc"
import "errors"
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
const TIMEOUT_MESSAGE string = "TIMEOUT";


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
var ClientArray []*Client;
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

//checks to see if a room with the given name exists in the RoomArray, if it does return it, if not return nil
func getRoomByName(roomName string) *Room{
  for _, room := range RoomArray{
    if room.name == roomName{
      return room;
    }
  }
  return nil;
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
        break //we want to jump out so as not to break
	}
      //else don't do anything
    }
    time.Sleep(time.Minute)//sleep the loop to lower processing
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
  lastAccessTime time.Time;
}

func getClientByName(clientName string) *Client{
  for _, cli := range ClientArray{
    if cli.name == clientName{
      return cli;
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
    lastAccessTime: time.Now(),//sets the currentTime on the client, if difference between last access time andd now is > TIMEOUT_DURATION, then Timeclient
  }
  ClientArray = append(ClientArray, &cli);
  go cli.watchForTimeout();
  return cli.name
}

//this function loops and checks if the user has been active in the last TIMEOUT_DURATION if not, boot the user
func (cli *Client) watchForTimeout(){
  for time.Since(cli.lastAccessTime) < TIMEOUT_DURATION{
  //do nothing, the client is active
  }
  processTimeout(cli)
}

func (cli *Client) updateLastUsedTime(){
  cli.lastAccessTime = time.Now();
}
//adds message to the clients output channel, messages should be single line, NON delimited strings, that is the message should not include a new line
//the name of the sender will be added to the message to form a final message in the form of "sender says: message\n"
func (cli *Client) messageClientFromClient(message string, sender *Client){
  message = string(sender.name)+" says: "+message+"\n";
  fmt.Println("we here")
  cli.outputChannel <- message;
}

//without a client argument assumes the message is coming from the server
func (cli *Client) messageClientFromServer(message string){
  message = "Server says: "+message+"\n";
  cli.outputChannel <- message;
}

//removes the client from his current room and takes the client out of the rooms list of users
func (cli *Client)removeClientFromCurrentRoom(){
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


//This function will remove the client from the Client Array, this function is intended to be used as part of the processQuitCommandHelper
func (client *Client)removeClientFromSystem(){
  //finds the client and removes them from the ClientArray
  for i,systemClients := range ClientArray{
    if client.name == systemClients.name {
      ClientArray = append(ClientArray[:i], ClientArray[i+1:]...)//deletes the element
    }
  }
  fmt.Println("there are currently: "+strconv.Itoa(len(ClientArray))+" clients connected");
}
/**********************************/

//wrapper for the internal function of the same name so that it can be used internally as well as by the client
func (server *Server)SendMessageToCurrentRoom(args *DoubleArgs, reply *string) error{
  sender := getClientByName(args.Arg1);
  sender.updateLastUsedTime();
  message := args.Arg2;
  sendMessageToCurrentRoom(sender, message);
  return nil;
}


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
  fmt.Println(room.clientList[0].currentRoom)
  for _, roomUser := range room.clientList {
    fmt.Println("looping room array user is: "+roomUser.name)
    //check to see if the user is currently active in the room
    if ((roomUser.currentRoom.name == room.name)) {
      go roomUser.messageClientFromClient(chatMessage.message, chatMessage.client)
    }
  }
  //save the message into the array of the rooms messages
  room.chatLog = append(room.chatLog, chatMessage);
}


//*****************RPC SERVER OBJECT************************
//here we will create a server object and expose the processing commands to the client, this way the client will be able to
//directly call the below functions

type Server int;

type DoubleArgs struct{
  Arg1 string;
  Arg2 string;
}

//the client must call this one time, it will set the client up on the server and send the server back their unique name
func (t *Server)Connect(name string, userNameReply *string) error {
    if len(ClientArray) < MAX_CLIENTS{//server can have more clients
      *userNameReply = addClient();
      return nil;
    }else{
      return errors.New("Server is currently full, try again later");
    }
}


//passes the latest message on the clients output channel to the client
func (server *Server) MessageClient(clientName string, reply *string) error {
cli := getClientByName(clientName);
fmt.Println("hoboutere"+clientName)
*reply = <-cli.outputChannel;
fmt.Println("hoboutere2"+clientName)
if (*reply == TIMEOUT_MESSAGE){
  return errors.New("You have timed out")
}
return nil;
}

//creates a room and logs to the console
func (server *Server)ProcessCreateRoomCommand(c *DoubleArgs, reply *string) error {
  client := getClientByName(c.Arg1);
  client.updateLastUsedTime();
  roomName := c.Arg2;
  room := createRoom(roomName, client);
  if room == nil { //name of room was not unique
    return nil
  }
  client.messageClientFromServer(client.name+" created a new room called "+roomName)
  return nil
}

func (server *Server)ProcessLeaveRoomCommand(clientName string, reply *string) error{
  client := getClientByName(clientName);
  client.updateLastUsedTime();
  client.removeClientFromCurrentRoom();
  client.messageClientFromServer("You have left the room.")
  return nil
}

//sends a list of the current users in the room to the client
func (server *Server)ProcessCurrRoomUsersCommand(clientName string, reply *string) error{
  //check if the user is in a room
  client := getClientByName(clientName);
  client.updateLastUsedTime();
  if client.currentRoom == nil{
    client.messageClientFromServer(NOT_IN_ROOM_ERR)
    return nil
  }
  client.messageClientFromServer("Current users in "+client.currentRoom.name+" are:")
  for _, users:= range client.currentRoom.clientList {
    client.messageClientFromServer(users.name);
  }
  return nil
}


//sends a message to the client telling them which room they are currently in, if not in a room, inform the user
 func (server *Server)ProcessCurrRoomCommand(clientName string, reply *string) error{
   client := getClientByName(clientName);
   client.updateLastUsedTime();
   if client.currentRoom == nil{
     client.messageClientFromServer(NOT_IN_ROOM_ERR)
     return nil
   }
   client.messageClientFromServer("current room: "+client.currentRoom.name);
   return nil
 }

//Loops through the HELP_INFO array and sends all the lines of help info to the user
func (server *Server)ProcessHelpCommand(clientName string, reply *string) error{
       client := getClientByName(clientName);
       client.updateLastUsedTime();
       for _, helpLine := range HELP_INFO{
         client.messageClientFromServer(helpLine);
       }
       return nil
}

//because processQuit is used elseware we are wrapping it in a serveer version
func (server *Server)ProcessQuitCommand(clientName string, reply *string) error{
  client := getClientByName(clientName);
  client.updateLastUsedTime();
  processQuitCommandHelper(client);
  return nil;
}
//quits the client from the server
func processQuitCommandHelper(client *Client){
  client.removeClientFromCurrentRoom();
  client.removeClientFromSystem();
}


func processTimeout(client *Client){
  client.outputChannel <- TIMEOUT_MESSAGE;
  fmt.Println(client.name+" timed out")
   processQuitCommandHelper(client)
}

//sends the list of rooms to the client
func (server *Server)ProcessListRoomsCommand(clientName string, reply *string) error{
  client := getClientByName(clientName);
  client.updateLastUsedTime();
  client.messageClientFromServer("List of rooms:")
  for _, roomName := range RoomArray{
    client.messageClientFromServer(roomName.name);
  }
  client.messageClientFromServer("");
  return nil;
}

//returns true of the room was joined successfully, returns false if there was a problem like the room does not exist
func (server *Server)ProcessJoinRoomCommand(args *DoubleArgs, reply *string) error{
  client := getClientByName(args.Arg1);
  client.updateLastUsedTime();
  roomName := args.Arg2;
  roomToJoin := getRoomByName(roomName);
  if roomToJoin == nil{ //the room doesnt exist
    fmt.Println(client.name+" tried to enter room: "+roomName+" which does not exist");
    client.messageClientFromServer("The room "+roomName+" does not exist")
    return nil;
  }
  //Room exists so now we can join it.
  //check if user is already in the room
  //add user to room if not in it already
  if roomToJoin.isClientInRoom(client) {
      client.messageClientFromServer("You are already in that room")
  } else {//join room and display all the rooms messages
    client.removeClientFromCurrentRoom();
    roomToJoin.clientList = append(roomToJoin.clientList, client);// add client to the rooms list
    //switch users current room to room
    client.currentRoom = roomToJoin;
    fmt.Println(client.name+" has joined room: "+client.currentRoom.name)
    sendMessageToCurrentRoom(client, CLIENT_JOINED_ROOM_MESSAGE)
    //display all messages in the room
    displayRoomsMessages(client, roomToJoin)
  }
  return nil
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
  rpc.Accept(ln)//continually waits on incomming commections
}
