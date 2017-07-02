package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/alimoeeny/gooauth2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/wangsongyan/wblog/helpers"
	"github.com/wangsongyan/wblog/models"
	"io/ioutil"
	"log"
	"net/http"
)

var oauthCfg = &oauth.Config{
	ClientId:     "25784931c6a043de301e",
	ClientSecret: "",
	AuthURL:      "https://github.com/login/oauth/authorize?client_id=%s&scope=user:email",
	TokenURL:     "https://github.com/login/oauth/access_token",
	RedirectURL:  "http://localhost:8090/oauth2callback",
	Scope:        "",
}

func SigninGet(c *gin.Context) {
	s := sessions.Default(c)
	if s.Get("User") != nil {
		c.Redirect(http.StatusMovedPermanently, "/admin/index")
	} else {
		c.HTML(http.StatusOK, "user/signin.html", gin.H{
			"authUrl": fmt.Sprintf(oauthCfg.AuthURL, oauthCfg.ClientId),
		})
	}
}

func SignupGet(c *gin.Context) {
	c.HTML(http.StatusOK, "user/signup.html", nil)
}

func LogoutGet(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	s.Save()
	c.Redirect(http.StatusMovedPermanently, "/signin")
}

func SignupPost(c *gin.Context) {
	email := c.PostForm("email")
	telephone := c.PostForm("telephone")
	password := c.PostForm("password")
	user := &models.User{
		Email:     email,
		Telephone: telephone,
		Password:  password,
	}
	var err error
	if len(user.Email) == 0 /*|| len(user.Telephone) == 0 */ || len(user.Password) == 0 {
		err = errors.New("error parameter.")
	} else {
		user.Password = helpers.Md5(user.Email + user.Password)
		err = user.Insert()
		if err == nil {
			c.HTML(http.StatusOK, "user/signin.html", gin.H{
				"user": user,
			})
			return
		} else {
			err = errors.New("email already exists.")
		}
	}
	c.HTML(http.StatusOK, "user/signup.html", gin.H{
		"message": err.Error(),
	})
}

func SigninPost(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	var err error
	if len(username) > 0 && len(password) > 0 {
		var user *models.User
		user, err = models.GetUserByUsername(username)
		if err == nil && user.Password == helpers.Md5(username+password) {
			s := sessions.Default(c)
			s.Set("UserID", user.ID)
			//s.Set("User", user)
			s.Save()
			c.Redirect(http.StatusMovedPermanently, "/admin/index")
			return
		} else {
			err = errors.New("invalid username or password.")
		}
	} else {
		err = errors.New("error parameter.")
	}
	c.HTML(http.StatusOK, "user/signin.html", gin.H{
		"message": err.Error(),
	})
}

func Oauth2Callback(c *gin.Context) {
	code := c.Query("code")
	t := &oauth.Transport{Config: oauthCfg}
	// Exchange the received code for a token
	tok, err := t.Exchange(code)
	if err == nil {
		tokenCache := oauth.CacheFile("./request.token")

		err := tokenCache.PutToken(tok)
		if err != nil {
			log.Fatal("Cache write:", err)
		}
		log.Printf("Token is cached in %v\n", tokenCache)
		token := tok.AccessToken
		fmt.Print(token)
		resp, err := http.Get("https://api.github.com/user?access_token=" + token)
		defer resp.Body.Close()
		if err != nil {
			log.Println(err)
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			var githubinfo struct {
				Login     string `json:"login"`
				AvatarUrl string `json:"avatar_url"`
			}
			fmt.Println(string(body))
			err = json.Unmarshal(body, &githubinfo)
			if err == nil {
				fmt.Println(githubinfo)
				var user *models.User
				sessionUser, exists := c.Get("User")
				fmt.Println(sessionUser)
				if exists {
					var ok bool
					user, ok = sessionUser.(*models.User)
					if ok {
						user.GithubLoginId = githubinfo.Login
						user.AvatarUrl = githubinfo.AvatarUrl
						err = user.Update()
						//TODO 检查githubId是否已经存在
					} else {
						err = errors.New("assert failed.")
					}
				} else {
					user := &models.User{
						Email:         githubinfo.Login,
						GithubLoginId: githubinfo.Login,
						AvatarUrl:     githubinfo.AvatarUrl,
					}
					user, err = user.FirstOrCreate()
				}

				if err == nil {
					s := sessions.Default(c)
					s.Set("UserID", user.ID)
					s.Save()
					c.Redirect(http.StatusMovedPermanently, "/admin/index")
					return
				} else {
					log.Print(err)
				}
			} else {
				log.Print(err)
			}
		}
	} else {
		log.Fatal(err)
	}
	c.Redirect(http.StatusMovedPermanently, "/signin")

}
