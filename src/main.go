package main

import (
	"github.com/gin-gonic/gin"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"log"
	"time"
	"strings"
	"strconv"
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"html"
)

type User struct
{
	ID int
	Name string
	Group int
	Is_Admin bool
	Session string
	Loggedin bool
}

type Page struct
{
	Title string
	Name string
	CurrentUser User
	ItemList map[int]interface{}
	StatusList []string
	Something interface{}
}

type Issue struct
{
	ID int
	Title string
	Content string
	CreatedBy int
	Status string
	Is_Closed bool
	tags string
	TagList []string
}

type IssueReply struct
{
	ID int
	ParentID int
	Content string
	CreatedBy int
	CreatedByName string
	CreatedAt int
}

var statusList []string = []string{"open","in-progress","resolved","wont-fix","not-a-bug","cannot-reproduce","confirmed","deferred","inefficient-to-implement"}
const hour int = 60 * 60
const day int = hour * 24
const month int = day * 30
const year int = day * 365
const saltLength int = 32
const sessionLength int = 80
var get_session_stmt *sql.Stmt
var create_issue_stmt *sql.Stmt
var create_issue_reply_stmt *sql.Stmt
var edit_issue_stmt *sql.Stmt
var edit_issue_reply_stmt *sql.Stmt
var delete_issue_reply_stmt *sql.Stmt
var login_stmt *sql.Stmt
var update_session_stmt *sql.Stmt
var logout_stmt *sql.Stmt
var set_password_stmt *sql.Stmt
var register_stmt *sql.Stmt
var username_exists_stmt *sql.Stmt
var custom_pages map[string]string = make(map[string]string)

func add_custom_page(path string, f os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	
	// Is this a directory..?
	fileInfo, err := os.Stat(path)
    is_dir := fileInfo.IsDir()
	if err != nil {
		return err
	}
	if is_dir {
		return err
	}
	
	custom_page, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	log.Print("Loaded the '" + path + "' page.")
	name := strings.TrimSuffix(path, filepath.Ext(path))
	custom_pages[name] = string(custom_page)
	return nil
}

func InternalError(err error, con *gin.Context, user User) {
	log.Fatal(err)
	var tList map[int]interface{}
	errmsg := "A problem has occured in the system."
	pi := Page{"Internal Server Error","error",user,tList,statusList,errmsg}
	con.HTML(500,"error.tmpl", pi)
}

func InternalErrorJSQ(err error, con *gin.Context, user User, is_js string) {
	log.Fatal(err)
	var tList map[int]interface{}
	errmsg := "A problem has occured in the system."
	if is_js == "0" {
		pi := Page{"Internal Server Error","error",user,tList,statusList,errmsg}
		con.HTML(500,"error.tmpl", pi)
	} else {
		con.JSON(500, gin.H{
			"errmsg": errmsg,
		})
	}
}

func LocalError(errmsg string, con *gin.Context, user User) {
	var tList map[int]interface{}
	pi := Page{"Local Error","error",user,tList,statusList,errmsg}
	con.HTML(500,"error.tmpl", pi)
}

func LocalErrorJSQ(errmsg string, con *gin.Context, user User, is_js string) {
	var tList map[int]interface{}
	if is_js == "0" {
		pi := Page{"Local Error","error",user,tList,statusList,errmsg}
		con.HTML(500,"error.tmpl", pi)
	} else {
		con.JSON(500, gin.H{
			"errmsg": errmsg,
		})
	}
}

func NoPermissionsJSQ(con *gin.Context, user User, is_js string) {
	errmsg := "You don't have permission to do that."
	var tList map[int]interface{}
	if is_js == "0" {
		pi := Page{"Local Error","error",user,tList,statusList,errmsg}
		con.HTML(500,"error.tmpl", pi)
	} else {
		con.JSON(500, gin.H{
			"errmsg": errmsg,
		})
	}
}

func CustomErrorJSQ(errmsg string, errcode int, errtitle string, con *gin.Context, user User, is_js string) {
	var tList map[int]interface{}
	if is_js == "0" {
		pi := Page{errtitle,"error",user,tList,statusList,errmsg}
		con.HTML(errcode,"error.tmpl", pi)
	} else {
		con.JSON(errcode, gin.H{
			"errmsg": errmsg,
		})
	}
}

// Generate a cryptographically secure set of random bytes..
func GenerateSafeString(length int) (string, error) {
	rb := make([]byte,length)
	_, err := rand.Read(rb)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(rb), nil
}

func SetPassword(uid int, password string) (error) {
	salt, err := GenerateSafeString(saltLength)
	if err != nil {
		return err
	}
	
	password = password + salt
	hashed_password, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	_, err = set_password_stmt.Exec(string(hashed_password), salt, uid)
	if err != nil {
		return err
	}
	return nil
}

func SessionCheck(con *gin.Context) (User) {
	user := User{0,"",0,false,"",false}
	var err error
	
	// Are there any session cookies..?
	// Assign it to user.name to avoid having to create a temporary variable for the type conversion
	user.Name, err = con.Cookie("uid")
	if err != nil {
		return user
	}
	user.ID, err = strconv.Atoi(user.Name)
	if err != nil {
		return user
	}
	user.Session, err = con.Cookie("session")
	if err != nil {
		return user
	}
	log.Print("ID: " + user.Name)
	log.Print("Session: " + user.Session)
	
	// Is this session valid..?
	err = get_session_stmt.QueryRow(user.ID,user.Session).Scan(&user.ID, &user.Name, &user.Group, &user.Is_Admin, &user.Session)
	if err == sql.ErrNoRows {
		log.Print("Couldn't find the user session")
		return user
	} else if err != nil {
		log.Fatal(err)
		return user
	}
	user.Loggedin = true
	log.Print("Logged in")
	log.Print("ID: " + strconv.Itoa(user.ID))
	log.Print("Group: " + strconv.Itoa(user.Group))
	log.Print("Name: " + user.Name)
	if user.Loggedin {
		log.Print("Loggedin: true")
	} else {
		log.Print("Loggedin: false")
	}
	if user.Is_Admin {
		log.Print("Is_Admin: true")
	} else {
		log.Print("Is_Admin: false")
	}
	log.Print("Session: " + user.Session)
	return user
}

func main(){
	//base_url := "http://localhost:8080"
	var err error
	user := "root"
	password := "password"
	if(password != ""){
		password = ":" + password
	}
	dbname := "bugz"
	db, err := sql.Open("mysql",user + password + "@tcp(127.0.0.1:3306)/" + dbname)
	if err != nil {
		log.Fatal(err)
	}
	
	// Make sure that the connection is alive..
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	
	log.Print("Preparing get_session statement.")
	get_session_stmt, err = db.Prepare("SELECT `uid`, `name`, `group`, `is_admin`, `session` FROM `users` WHERE `uid` = ? AND `session` = ? AND `session` <> ''")
	if err != nil {
		log.Fatal(err)
	}
	_ = get_session_stmt // Bug fix, compiler isn't recognising this despite it being used, probably because it's hidden behind if statements
	
	log.Print("Preparing create_issue statement.")
	create_issue_stmt, err = db.Prepare("INSERT INTO issues(title,content,createdAt,lastReplyAt,createdBy) VALUES(?,?,?,0,?)")
	if err != nil {
		log.Fatal(err)
	}
	
	log.Print("Preparing create_issue_reply statement.")
	create_issue_reply_stmt, err = db.Prepare("INSERT INTO issues_replies(iid,content,createdAt,createdBy) VALUES(?,?,?,?)")
	if err != nil {
		log.Fatal(err)
	}
	
	log.Print("Preparing edit_issue statement.")
	edit_issue_stmt, err = db.Prepare("UPDATE issues SET title = ?, status = ?, content = ? WHERE iid = ?")
	if err != nil {
		log.Fatal(err)
	}
	
	log.Print("Preparing edit_issue_reply statement.")
	edit_issue_reply_stmt, err = db.Prepare("UPDATE issues_replies SET content = ? WHERE irid = ?")
	if err != nil {
		log.Fatal(err)
	}
	
	log.Print("Preparing delete_issue_reply statement.")
	delete_issue_reply_stmt, err = db.Prepare("DELETE FROM issues_replies WHERE irid = ?")
	if err != nil {
		log.Fatal(err)
	}
	
	log.Print("Preparing login statement.")
	login_stmt, err = db.Prepare("SELECT `uid`, `name`, `password`, `salt` FROM `users` WHERE `name` = ?")
	if err != nil {
		log.Fatal(err)
	}
	
	log.Print("Preparing update_session statement.")
	update_session_stmt, err = db.Prepare("UPDATE users SET session = ? WHERE uid = ?")
	if err != nil {
		log.Fatal(err)
	}
	
	log.Print("Preparing logout statement.")
	logout_stmt, err = db.Prepare("UPDATE users SET session = '' WHERE uid = ?")
	if err != nil {
		log.Fatal(err)
	}
	
	log.Print("Preparing set_password statement.")
	set_password_stmt, err = db.Prepare("UPDATE users SET password = ?, salt = ? WHERE uid = ?")
	if err != nil {
		log.Fatal(err)
	}
	
	// Add an admin version of register_stmt with more flexibility
	// create_account_stmt, err = db.Prepare("INSERT INTO 
	
	register_stmt, err = db.Prepare("INSERT INTO users(`name`,`password`,`salt`,`group`,`is_admin`,`session`) VALUES(?,?,?,0,0,?)")
	if err != nil {
		log.Fatal(err)
	}
	
	log.Print("Preparing get_session statement.")
	username_exists_stmt, err = db.Prepare("SELECT `name` FROM `users` WHERE `name` = ?")
	if err != nil {
		log.Fatal(err)
	}
	
	log.Print("Loading the custom pages.")
	err = filepath.Walk("pages/", add_custom_page)
	if err != nil {
		log.Fatal(err)
	}
	
	router := gin.Default()
	// In a directory to stop it clashing with the other paths
	router.Static("/static","./public")
	router.LoadHTMLGlob("templates/*")
	
	// GET functions
	overview := func(con *gin.Context){
		user := SessionCheck(con)
		var tList map[int]interface{} 
		pi := Page{"Overview","overview",user,tList,statusList,0}
		con.HTML(200,"overview.tmpl", pi)
	}
	
	custom_page := func(con *gin.Context){
		user := SessionCheck(con)
		name := con.Param("name")
		
		val, ok := custom_pages[name];
		if ok {
			var tList map[int]interface{} 
			pi := Page{"Page","page",user,tList,statusList,val}
			con.HTML(200,"custom_page.tmpl", pi)
		} else {
			var tList map[int]interface{}
			errmsg := "The requested page doesn't exist."
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(404,"error.tmpl", pi)
			return
		}
	}
	
	issues := func(con *gin.Context){
		user := SessionCheck(con)
		var(
			issueList map[int]interface{}
			currentID int
			
			iid int
			title string
			content string
			createdBy int
			status string
			is_closed bool
			tags string
		)
		issueList = make(map[int]interface{})
		currentID = 0
		
		rows, err := db.Query("select iid, title, content, createdBy, status, is_closed, tags from issues")
		if err != nil {
			InternalError(err,con,user)
			return
		}
		defer rows.Close()
		
		for rows.Next() {
			err := rows.Scan(&iid, &title, &content, &createdBy, &status, &is_closed, &tags)
			if err != nil {
				InternalError(err,con,user)
				return
			}
			issueList[currentID] = Issue{iid,title,content,createdBy, status, is_closed, tags, strings.Split(tags," ")}
			currentID++
		}
		err = rows.Err()
		if err != nil {
			InternalError(err,con,user)
			return
		}
		pi := Page{"Issue List","issues",user,issueList,statusList,0}
		con.HTML(200,"issues.tmpl", pi)
	}
	
	issue_id := func(con *gin.Context){
		user := SessionCheck(con)
		var(
			iid int
			irid int
			title string
			content string
			createdBy int
			createdByName string
			replyContent string
			replyCreatedBy int
			replyCreatedByName string
			replyCreatedAt int
			status string
			is_closed bool
			tags string
			
			currentID int
			replyList map[int]interface{}
		)
		replyList = make(map[int]interface{})
		currentID = 0
		
		iid, err := strconv.Atoi(con.Param("id"))
		if err != nil {
			LocalError("The provided IssueID is not a valid number.",con,user)
			return
		}
		
		// Get the issue..
		//err = db.QueryRow("select title, content, createdBy, status, is_closed, tags from issues where iid = ?", iid).Scan(&title, &content, &createdBy, &status, &is_closed, &tags)
		err = db.QueryRow("select issues.title, issues.content, issues.createdBy, issues.status, issues.is_closed, issues.tags, users.name from issues left join users ON issues.createdBy = users.uid where iid = ?", iid).Scan(&title, &content, &createdBy, &status, &is_closed, &tags, &createdByName)
		if err == sql.ErrNoRows {
			var tList map[int]interface{}
			errmsg := "The requested issue doesn't exist."
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(404,"error.tmpl", pi)
			return
		} else if err != nil {
			InternalError(err,con,user)
			return
		}
		
		var tdata map[string]string
		tdata = make(map[string]string)
		tdata["iid"] = strconv.Itoa(iid)
		tdata["title"] = title
		tdata["content"] = content
		tdata["status"] = status
		//tdata["createdBy"] = string(createdBy)
		tdata["createdByName"] = createdByName
		
		// Get the replies..
		//rows, err := db.Query("select irid, content, createdBy, createdAt from issues_replies where iid = ?", iid)
		rows, err := db.Query("select issues_replies.irid, issues_replies.content, issues_replies.createdBy, issues_replies.createdAt, users.name from issues_replies left join users ON issues_replies.createdBy = users.uid where iid = ?", iid)
		if err != nil {
			InternalError(err,con,user)
			return
		}
		defer rows.Close()
		
		for rows.Next() {
			err := rows.Scan(&irid, &replyContent, &replyCreatedBy, &replyCreatedAt, &replyCreatedByName)
			if err != nil {
				log.Fatal(err)
			}
			//replyList[currentID] = Issue{irid,"",replyContent,createdBy, status, is_closed, tags, strings.Split(tags," ")}
			replyList[currentID] = IssueReply{irid,iid,replyContent,replyCreatedBy,replyCreatedByName,replyCreatedAt}
			currentID++
		}
		err = rows.Err()
		if err != nil {
			InternalError(err,con,user)
			return
		}
		
		pi := Page{title,"issue",user,replyList,statusList,tdata}
		con.HTML(200,"issue.tmpl", pi)
	}
	
	issue_create := func(con *gin.Context){
		user := SessionCheck(con)
		var tList map[int]interface{} 
		pi := Page{"Create Issue","create-issue",user,tList,statusList,0}
		con.HTML(200,"create-issue.tmpl", pi)
	}
	
	// POST functions. Authorised users only.
	create_issue := func(con *gin.Context) {
		user := SessionCheck(con)
		if !user.Loggedin {
			var tList map[int]interface{}
			errmsg := "You need to login to create issues."
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
			return
		}
		success := 1
		
		res, err := create_issue_stmt.Exec(html.EscapeString(con.PostForm("issue-name")),html.EscapeString(con.PostForm("issue-content")),int32(time.Now().Unix()),user.ID)
		if err != nil {
			log.Print(err)
			success = 0
		}
		
		lastId, err := res.LastInsertId()
		if err != nil {
			log.Print(err)
			success = 0
		}
		
		if success != 1 {
			var tList map[int]interface{}
			errmsg := "Unable to create the issue"
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
		} else {
			con.Redirect(301, "/issue/" + strconv.FormatInt(lastId, 10))
		}
	}
	
	create_reply := func(con *gin.Context) {
		var iid int
		user := SessionCheck(con)
		if !user.Loggedin {
			var tList map[int]interface{}
			errmsg := "You need to login to create replies."
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
			return
		}
		
		success := 1
		iid, err = strconv.Atoi(con.PostForm("iid"))
		if err != nil {
			log.Print(err)
			success = 0
			
			var tList map[int]interface{}
			errmsg := "Unable to create the issue reply"
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
			return
		}
		//log.Println("An issue reply is being created")
		
		_, err := create_issue_reply_stmt.Exec(iid,html.EscapeString(con.PostForm("reply-content")),int32(time.Now().Unix()),user.ID)
		if err != nil {
			log.Print(err)
			success = 0
		}
		
		if success != 1 {
			var tList map[int]interface{}
			errmsg := "Unable to create the issue reply"
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
		} else {
			con.Redirect(301, "/issue/" + strconv.Itoa(iid))
		}
	}
	
	edit_issue := func(con *gin.Context) {
		is_js := con.DefaultPostForm("issue_js","0")
		user := SessionCheck(con)
		
		if !user.Is_Admin {
			NoPermissionsJSQ(con, user, is_js)
			return
		}
		
		var iid int
		iid, err := strconv.Atoi(con.Param("id"))
		if err != nil {
			LocalErrorJSQ("The provided IssueID is not a valid number.",con,user,is_js)
			return
		}
		
		issue_name := con.PostForm("issue_name")
		issue_status := con.PostForm("issue_status")
		issue_content := html.EscapeString(con.PostForm("issue_content"))
		_, err = edit_issue_stmt.Exec(issue_name, issue_status, issue_content, iid)
		if err != nil {
			InternalErrorJSQ(err,con,user,is_js)
			return
		}
		
		if is_js == "0" {
			con.Redirect(301, "/issue/" + strconv.Itoa(iid))
		} else {
			con.JSON(200, gin.H{
				"success": "1",
			})
		}
	}
	
	issue_reply_edit_submit := func(con *gin.Context) {
		is_js := con.DefaultPostForm("issue_js","0")
		user := SessionCheck(con)

		if !user.Is_Admin {
			NoPermissionsJSQ(con, user, is_js)
			return
		}
		
		irid, err := strconv.Atoi(con.Param("id"))
		if err != nil {
			LocalError("The provided Issue Reply ID is not a valid number.",con,user)
			return
		}
		
		content := html.EscapeString(con.PostForm("edit_item"))
		_, err = edit_issue_reply_stmt.Exec(content, irid)
		if err != nil {
			InternalError(err,con,user)
			return
		}
		
		// Get the Issue ID..
		var iid int
		err = db.QueryRow("select iid from issues_replies where irid = ?", irid).Scan(&iid)
		if err != nil {
			InternalError(err,con,user)
			return
		}
		
		if is_js == "0" {
			con.Redirect(301, "/issue/" + strconv.Itoa(iid) + "#reply-" + strconv.Itoa(irid))
		} else {
			con.JSON(200, gin.H{
				"success": "1",
			})
		}
	}
	
	issue_reply_delete_submit := func(con *gin.Context) {
		is_js := con.DefaultPostForm("is_js","0")
		user := SessionCheck(con)

		if !user.Is_Admin {
			NoPermissionsJSQ(con, user, is_js)
			return
		}
		
		irid, err := strconv.Atoi(con.Param("id"))
		if err != nil {
			LocalErrorJSQ("The provided Issue Reply ID is not a valid number.",con,user,is_js)
			return
		}
		
		var iid int
		err = db.QueryRow("SELECT iid from issues_replies where irid = ?", irid).Scan(&iid)
		if err == sql.ErrNoRows {
			LocalErrorJSQ("The issue reply you tried to delete doesn't exist.",con,user,is_js)
			return
		} else if err != nil {
			InternalErrorJSQ(err,con,user,is_js)
			return
		}
		
		_, err = delete_issue_reply_stmt.Exec(irid)
		if err != nil {
			InternalErrorJSQ(err,con,user,is_js)
			return
		}
		log.Print("The issue reply '" + strconv.Itoa(irid) + "' was deleted by User ID #" + strconv.Itoa(user.ID) + ".")
		
		if is_js == "0" {
			//con.Redirect(301, "/issue/" + strconv.Itoa(iid))
		} else {
			con.JSON(200, gin.H{
				"success": "1",
			})
		}
	}
	
	account_own_edit_critical := func(con *gin.Context) {
		user := SessionCheck(con)
		if !user.Loggedin {
			var tList map[int]interface{}
			errmsg := "You need to login to edit your own account."
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
			return
		}
		
		var tList map[int]interface{} 
		pi := Page{"Edit Password","account-own-edit",user,tList,statusList,0}
		con.HTML(200,"account-own-edit.tmpl", pi)
	}
	
	account_own_edit_critical_submit := func(con *gin.Context) {
		user := SessionCheck(con)
		if !user.Loggedin {
			var tList map[int]interface{}
			errmsg := "You need to login to edit your own account."
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
			return
		}
		
		current_password, err := strconv.Atoi(con.Param("account-current-password"))
		new_password, err := strconv.Atoi(con.Param("account-new-password"))
		confirm_password, err := strconv.Atoi(con.Param("account-confirm-password"))
		
		
		
		var tList map[int]interface{} 
		pi := Page{"Edit Password","account-own-edit",user,tList,statusList,0}
		con.HTML(200,"account-own-edit.tmpl", pi)
	}
	
	logout := func(con *gin.Context) {
		user := SessionCheck(con)
		if !user.Loggedin {
			var tList map[int]interface{}
			errmsg := "You can't logout without logging in first."
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
			return
		}
		
		_, err := logout_stmt.Exec(user.ID)
		if err != nil {
			InternalError(err,con,user)
			return
		}
		con.Redirect(301,"/")
	}
	
	login := func(con *gin.Context) {
		user := SessionCheck(con)
		if user.Loggedin {
			var tList map[int]interface{}
			errmsg := "You're already logged in."
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
			return
		}
		
		var tList map[int]interface{} 
		pi := Page{"Login","login",user,tList,statusList,0}
		con.HTML(200,"login.tmpl", pi)
	}
	
	login_submit := func(con *gin.Context) {
		user := SessionCheck(con)
		if user.Loggedin {
			var tList map[int]interface{}
			errmsg := "You're already logged in."
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
			return
		}
		
		var uid int
		var real_password string
		var salt string
		var session string
		username := html.EscapeString(con.PostForm("username"))
		password := con.PostForm("password")
		
		err = login_stmt.QueryRow(username).Scan(&uid, &username, &real_password, &salt)
		if err == sql.ErrNoRows {
			var tList map[int]interface{}
			errmsg := "That username doesn't exist."
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
			return
		} else if err != nil {
			InternalError(err,con,user)
			return
		}
		
		// Emergency password reset mechanism..
		if salt == "" {
			if password != real_password {
				var tList map[int]interface{}
				errmsg := "That's not the correct password."
				pi := Page{"Error","error",user,tList,statusList,errmsg}
				con.HTML(500,"error.tmpl", pi)
				return
			}
			
			// Re-encrypt the password
			SetPassword(uid, password)
		} else { // Normal login..
			password = password + salt
			//hashed_password, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				InternalError(err,con,user)
				return
			}
			
			//log.Print("Hashed: " + string(hashed_password))
			//log.Print("Real:   " + real_password)
			//if string(hashed_password) != real_password {
			err := bcrypt.CompareHashAndPassword([]byte(real_password), []byte(password))
			if err == bcrypt.ErrMismatchedHashAndPassword {
				var tList map[int]interface{}
				errmsg := "That's not the correct password."
				pi := Page{"Error","error",user,tList,statusList,errmsg}
				con.HTML(500,"error.tmpl", pi)
				return
			} else if err != nil {
				InternalError(err,con,user)
				return
			}
		}
		
		session, err := GenerateSafeString(sessionLength)
		if err != nil {
			InternalError(err,con,user)
			return
		}
		
		_, err = update_session_stmt.Exec(session, user.ID)
		if err != nil {
			InternalError(err,con,user)
			return
		}
		
		log.Print("Successful Login")
		con.SetCookie("uid",strconv.Itoa(uid),year,"/","",false,true)
		con.SetCookie("session",session,year,"/","",false,true)
		con.Redirect(301,"/")
	}
	
	register := func(con *gin.Context) {
		user := SessionCheck(con)
		if user.Loggedin {
			var tList map[int]interface{}
			errmsg := "You're already logged in."
			pi := Page{"Error","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
			return
		}
		
		var tList map[int]interface{} 
		pi := Page{"Registration","register",user,tList,statusList,0}
		con.HTML(200,"register.tmpl", pi)
	}
	
	register_submit := func(con *gin.Context) {
		user := SessionCheck(con)
		username := html.EscapeString(con.PostForm("username"))
		password := con.PostForm("password")
		confirm_password := con.PostForm("confirm_password")
		log.Print("Registration Attempt! Username: " + username)
		
		// Do the two inputted passwords match..?
		if password != confirm_password {
			var tList map[int]interface{}
			errmsg := "The two passwords don't match."
			pi := Page{"Password Mismatch","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
			return
		}
		
		// Is this username already taken..?
		err = username_exists_stmt.QueryRow(username).Scan(&username)
		if err != nil && err != sql.ErrNoRows {
			InternalError(err,con,user)
			return
		} else if err != sql.ErrNoRows {
			var tList map[int]interface{}
			errmsg := "This username isn't available. Try another."
			pi := Page{"Username Taken","error",user,tList,statusList,errmsg}
			con.HTML(500,"error.tmpl", pi)
			return
		}
		
		salt, err := GenerateSafeString(saltLength)
		if err != nil {
			InternalError(err,con,user)
			return
		}
		
		session, err := GenerateSafeString(sessionLength)
		if err != nil {
			InternalError(err,con,user)
			return
		}
		
		password = password + salt
		hashed_password, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			InternalError(err,con,user)
			return
		}
		
		res, err := register_stmt.Exec(username,string(hashed_password),salt,session)
		if err != nil {
			InternalError(err,con,user)
			return
		}
		
		lastId, err := res.LastInsertId()
		if err != nil {
			InternalError(err,con,user)
			return
		}
		
		con.SetCookie("uid",strconv.FormatInt(lastId, 10),year,"/","",false,true)
		con.SetCookie("session",session,year,"/","",false,true)
		con.Redirect(301,"/")
	}
	
	// Issues
	router.GET("/", issues)
	router.GET("/overview", overview)
	router.GET("/issues", issues)
	router.GET("/issues/create", issue_create)
	router.GET("/issue/:id", issue_id)
	router.POST("/issue/create/submit", create_issue)
	router.POST("/reply/create", create_reply)
	//router.POST("/reply/edit/:id", issue_reply_edit)
	//router.POST("/reply/delete/:id", issue_reply_delete)
	router.POST("/reply/edit/submit/:id", issue_reply_edit_submit)
	router.POST("/reply/delete/submit/:id", issue_reply_delete_submit)
	router.POST("/issue/edit/submit/:id", edit_issue)
	
	// Custom Pages
	router.GET("/pages/:name", custom_page)
	
	// Accounts
	router.GET("/accounts/login", login)
	router.GET("/accounts/create", register)
	router.GET("/accounts/logout", logout)
	router.POST("/accounts/login/submit", login_submit)
	router.POST("/accounts/create/submit", register_submit)
	
	//router.GET("/accounts/list", login) // Redirect /accounts/ and /user/ to here..
	//router.GET("/accounts/create/full", logout)
	//router.GET("/user/edit", logout)
	router.GET("/user/edit/critical", account_own_edit_critical) // Password & Email
	router.GET("/user/edit/critical/submit", account_own_edit_critical_submit)
	//router.GET("/user/:id/edit", logout)
	//router.GET("/user/:id/ban", logout)
	
	defer db.Close()
	router.Run(":8080")
}