package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var Db *sql.DB

func init() {
	var err error
	fmt.Println("connecting to mysql")
	Db, err = sql.Open("mysql", "root:111111@tcp(172.26.108.113:3306)/go")

	err = Db.Ping()
	if err != nil {
		fmt.Println("connect error")
	} else {
		fmt.Println("connected to mysql")
	}

	Db.SetMaxOpenConns(10)
	Db.SetMaxIdleConns(10)

}

func main() {

	// Init()
	r := gin.Default()

	//允许跨域访问
	r.Use(CrosHandler())

	//设置分组路由
	v1 := r.Group("/user")

	//根据分组设置路由
	{
		v1.POST("/login", Login)
		v1.POST("/register", Register)
		v1.GET("/self", Self)
		v1.POST("/logout", Logout)
	}

	//启动
	r.Run() // listen and serve on 0.0.0.0:8080

}

var (
	Secret     = "service-computing-blogs"
	ExpireTime = 3600
)

type JWTClaims struct {
	jwt.StandardClaims
	UserId   int64  `json:"userid"`
	Password string `json:"password"`
	UserName string `json:"username"`
	Email    string `json:"email"`
}

func getToken(claims *JWTClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(Secret))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func verifyToken(strToken string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(strToken, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(Secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, err
	}
	if err := token.Claims.Valid(); err != nil {
		return nil, err
	}
	return claims, nil
}

type registerModel struct {
	UserName string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

func Register(c *gin.Context) {
	var registerInfo registerModel
	c.Bind(&registerInfo)

	result, err := Db.Exec("insert into user (username, password, email) values (?,?,?);", registerInfo.UserName, registerInfo.Password, registerInfo.Email)
	var id int64
	if err != nil {
		fmt.Println("err:%s", err)
	} else {
		id, _ = result.LastInsertId()
	}

	claims := &JWTClaims{
		UserId:   id,
		UserName: registerInfo.UserName,
		Password: registerInfo.Password,
		Email:    registerInfo.Email,
	}
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(time.Second * time.Duration(ExpireTime)).Unix()
	signedToken, err := getToken(claims)
	c.SetCookie("jwt-token", signedToken, 3600, "/", ".", false, true)
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

type loginModel struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

func Logout(c *gin.Context) {
	c.SetCookie("jwt-token", "", 0, "/", ".", false, true)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

func Login(c *gin.Context) {
	var loginInfo loginModel
	c.Bind(&loginInfo)

	var id int64
	var email string
	status := "not defined"
	err := Db.QueryRow("select user_id, email from user where username = ? and password = ?", loginInfo.UserName, loginInfo.Password).Scan(&id, &email)
	if err != nil {
		if err == sql.ErrNoRows { //如果未查询到对应字段则...
			status = "not found"
			fmt.Println("not found")
		} else {
			status = "failure"
			fmt.Println("failue")
			log.Fatal(err)
		}
	} else {
		status = "success"
	}

	claims := &JWTClaims{
		UserId:   id,
		UserName: loginInfo.UserName,
		Password: loginInfo.Password,
		Email:    email,
	}
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = time.Now().Add(time.Second * time.Duration(ExpireTime)).Unix()
	signedToken, _ := getToken(claims)
	c.SetCookie("jwt-token", signedToken, 3600, "/", ".", false, true)

	c.JSON(http.StatusOK, gin.H{
		"status": status,
	})
}

func Self(c *gin.Context) {
	strToken, err := c.Cookie("jwt-token")
	claims, err := verifyToken(strToken)
	if err != nil {
		c.String(401, err.Error())
		return
	}
	claims.ExpiresAt = time.Now().Unix() + (claims.ExpiresAt - claims.IssuedAt)
	signedToken, err := getToken(claims)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	c.SetCookie("jwt-token", signedToken, 3600, "/", ".", false, true)

	c.JSON(http.StatusOK, gin.H{
		"name":  claims.UserName,
		"email": claims.Email,
		"id":    claims.UserId,
	})
}

//跨域访问：cross  origin resource share
func CrosHandler() gin.HandlerFunc {
	return func(context *gin.Context) {
		method := context.Request.Method
		context.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		context.Header("Access-Control-Allow-Origin", "*") // 设置允许访问所有域
		context.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE")
		context.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session,X_Requested_With,Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma,token,openid,opentoken")
		context.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar")
		context.Header("Access-Control-Max-Age", "172800")
		context.Header("Access-Control-Allow-Credentials", "false")
		context.Set("content-type", "application/json") //设置返回格式是json

		if method == "OPTIONS" {
			context.JSON(http.StatusOK, "OK")
		}

		//处理请求
		context.Next()
	}
}
