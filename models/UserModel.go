package models

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/validation"
	"github.com/go-pay/util"
	"log"
	"strconv"
	"time"
)

// 用户表
type User struct {
	Id            int64
	Username      string `json:"name",orm:"unique"` // 唯一。这个拼音的简写
	Nickname      string //中文名，注意这里，很多都要查询中文名才行`orm:"unique;size(32)" form:"Nickname" valid:"Required;MaxSize(20);MinSize(2)"`
	Password      string `orm:"size(50)"`
	Repassword    string `form:"Repassword"`
	Email         string `orm:"size(32)"`
	Sex           string `orm:"size(4)"`
	IsPartyMember bool   `json:"IsPartyMember"`
	Department    string //分院
	Secoffice     string //科室,这里应该用科室id，才能保证即时重名也不怕。否则，查看科室必须要上溯到分院才能避免科室名称重复问题
	Remark        string `orm:"null;size(200)"`
	Ip            string //ip地址
	Port          string
	Status        int       `orm:"default(1)"`
	Lastlogintime time.Time `orm:"type(datetime);auto_now_add"`
	Createtime    time.Time `orm:"type(datetime);auto_now_add" `
	Updated       time.Time `orm:"type(datetime);auto_now_add" `
	Role          string    `json:"role",orm:"default('4')"` //这个不是角色，这个无意义
	// Salt          string    `orm:"size(50)"`
	// Roles         []*Role   `orm:"rel(m2m)"`
}

// 用户和openid对应表,一个用户对应多个openid
type UserOpenID struct {
	Id         int64
	Uid        int64
	OpenID     string
	Createtime time.Time `orm:"null;type(datetime);auto_now_add" `
}

// 用户和AvatorUrl对应表,一个用户对应多个AvatorUrl
type UserAvatar struct {
	Id         int64
	Uid        int64
	AvatarUrl  string
	Createtime time.Time `orm:"type(datetime);auto_now_add" `
}

// 用户和AppreciationUrl对应表,一个用户对应多个AppreciationUrl
type UserAppreciation struct {
	Id              int64
	Uid             int64
	AppreciationUrl string
	Createtime      time.Time `orm:"type(datetime);auto_now_add" `
}

// Id            int64
// Username      string    `orm:"unique;size(32)" form:"Username"  valid:"Required;MaxSize(20);MinSize(6)"`
// Password      string    `orm:"size(32)" form:"Password" valid:"Required;MaxSize(20);MinSize(6)"`

func init() { //
	orm.RegisterModel(new(User), new(UserOpenID), new(UserAvatar), new(UserAppreciation))
	// _db.CreateTable(&User{})
}

// 这个是使用的，下面那个adduser不知干啥的
func SaveUser(user User) (uid int64, err error) {
	o := orm.NewOrm()
	var user1 User
	//判断是否有重名
	err = o.QueryTable("user").Filter("username", user.Username).One(&user1, "Id")
	if err == orm.ErrNoRows { //Filter("tnumber", tnumber).One(topic, "Id")==nil则无法建立
		// 没有找到记录
		uid, err2 := o.Insert(&user)
		if err2 == nil {
			return uid, err2
		} else {
			return 0, err2
		}
	} //else { //应该进行更新操作
	// user1 := &User{Id: user1.Id}
	// 	user1.Username = user.Username
	// 	user1.Nickname = user.Nickname
	// 	user1.Password = user.Password
	// 	user1.Repassword = user.Repassword
	// 	user1.Email = user.Email
	// 	user1.Department = user.Department
	// 	user1.Secoffice = user.Secoffice
	// 	// user1.Remark = user.Remark
	// 	user1.Ip = user.Ip
	// 	user1.Status = user.Status
	// 	user1.Lastlogintime = user.Lastlogintime
	// 	user1.Createtime = time.Now()
	// 	user1.Role = user.Role
	// 	_, err = o.Update(&user1)
	// 	if err != nil {
	// 		return 0, err
	// 	}
	// 	uid = user1.Id
	//return 0, err
	//}
	return 0, err
}

// 后台手工操作添加微信小程序openid和用户名
func AddUserOpenID(userid int64, openid string) (id int64, err error) {
	o := orm.NewOrm()
	var useropenid UserOpenID
	useropenid.Uid = userid
	useropenid.OpenID = openid
	//判断是否有重名
	err = o.QueryTable("UserOpenID").Filter("open_i_d", openid).One(&useropenid, "Id")
	if err == orm.ErrNoRows { //Filter("tnumber", tnumber).One(topic, "Id")==nil则无法建立
		// 没有找到记录
		id, err = o.Insert(&useropenid)
		if err != nil {
			return id, err
		}
	} else {
		// 20250420如果openid存在，进行更新openid对应用户名。所以openid永远对应最新登录的用户名。
		_, err = o.Update(&useropenid, "Uid")
		if err != nil {
			return useropenid.Uid, err
		}
	}
	return useropenid.Uid, err //这里需要改改，否则，即使已经存在，则err为空。id=0则已经存在
}

// 取出所有openid
func GetOpenIDs() (openids []*UserOpenID, err error) {
	o := orm.NewOrm()
	// openid := new(UserOpenID)
	qs := o.QueryTable("UserOpenID")
	_, err = qs.All(&openids)
	if err != nil {
		return nil, err
	}
	// count, _ = qs.Count()
	return openids, err
}

// 添加用户头像
func AddUserAvator(userid int64, avatarurl string) (id int64, err error) {
	o := orm.NewOrm()
	var useravatar UserAvatar
	// 没有找到记录
	useravatar.Uid = userid
	useravatar.AvatarUrl = avatarurl
	id, err = o.Insert(&useravatar)
	if err != nil {
		return id, err
	}
	return id, err //这里需要改改，否则，即使已经存在，则err为空。
}

// 添加用户赞赏码
func AddUserAppreciation(userid int64, appreciationurl string) (id int64, err error) {
	o := orm.NewOrm()
	var userappreciation UserAppreciation
	// 没有找到记录
	userappreciation.Uid = userid
	userappreciation.AppreciationUrl = appreciationurl
	id, err = o.Insert(&userappreciation)
	if err != nil {
		return id, err
	}
	return id, err //这里需要改改，否则，即使已经存在，则err为空。
}

// 根据openid查user
func GetUserByOpenID(openid string) (user User, err error) {
	o := orm.NewOrm()
	// var useropenid UserOpenID
	qs := o.QueryTable("UserOpenID")

	// // user := User{Name: "slene"}
	useropenid := UserOpenID{OpenID: openid}
	// // Three return values：Is Created，Object Id，Error
	// if created, id, err := o.ReadOrCreate(&useropenid, "OpenId"); err == nil {
	// 	if created {
	// 		fmt.Println("New Insert an object. Id:", id)
	// 	} else {
	// 		fmt.Println("Get an object. Id:", id)
	// 	}
	// }

	// 查出第一个openid
	err = qs.Filter("open_i_d", openid).One(&useropenid)
	if err != nil {
		return user, err
	}
	// 如果不存在，则新建openid

	// 查询出user所有信息
	user = User{Id: useropenid.Uid}
	o.Read(&user) //这里是默认主键查询。=(&user,"Id")
	// 如果不存在，则随机存入用户名

	return user, err
}

// 根据openid插入或查询user
func GetwxUserByOpenID(openid, nickname string) (user User, id int64, err error) {
	// db := _db //GetDB()
	o := orm.NewOrm()
	qs := o.QueryTable("user") //不知道主键就用这个过滤操作
	qs2 := o.QueryTable("UserOpenID")
	var useropenid UserOpenID
	err = qs.Filter("nickname", nickname).One(&user)
	// 没找到记录err == orm.ErrNoRows，或找到多条记录err == orm.ErrMultiRows
	err2 := qs2.Filter("open_i_d", openid).One(&useropenid)
	//这里要加密
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(util.RandomString(6)))
	cipherStr := md5Ctx.Sum(nil)

	// 没找到nickname也没找到openid，则新建
	if err != nil && err2 != nil {
		// 新建用户和openid
		user.Nickname = nickname
		user.Username = util.RandomString(6)
		user.Status = 1
		user.Password = hex.EncodeToString(cipherStr)
		id, err := o.Insert(&user)
		if err != nil {
			return user, id, err
		}
		// 新建openid
		useropenid.Uid = id
		useropenid.OpenID = openid
		id2, err := o.Insert(&useropenid)
		if err != nil {
			return user, id2, err
		}
		return user, id, nil
	} else if err == nil && err2 != nil {
		// 找到nickname，没找到openid，则用nickname的id来新建openid，同时也要查询username是否是空，空则随机赋值
		// 新建openid
		useropenid.Uid = user.Id
		useropenid.OpenID = openid
		id2, err := o.Insert(&useropenid)
		if err != nil {
			return user, id2, err
		}
		if user.Username == "" {
			user.Username = util.RandomString(6)
			_, err := o.Update(&user, "Username", "Updated")
			if err != nil {
				return user, user.Id, err
			}
		}
		return user, user.Id, nil
	} else if err == nil && err2 == nil {
		// 找到nickname，找到openid
		if user.Username == "" {
			user.Username = util.RandomString(6)
			_, err := o.Update(&user, "Username", "Updated")
			if err != nil {
				return user, user.Id, err
			}
		}
		// 返回user
		return user, user.Id, nil
	} else if err != nil && err2 == nil {
		// 没找到nickname，找到openid
		// 查出openid对应的用户，
		err = qs.Filter("id", useropenid.Uid).One(&user)
		if err != nil {
			// return user, err
			// 新建用户表，随机username
			user.Nickname = nickname
			user.Username = util.RandomString(6)
			user.Status = 1
			user.Password = hex.EncodeToString(cipherStr)
			id, err := o.Insert(&user)
			if err != nil {
				return user, id, err
			}
		}
		return user, user.Id, nil
	}
	return user, id, err
}

// 用户修改自己的某个字段
func UpdateUser(cid int64, fieldname, value string) error {
	o := orm.NewOrm()
	qs := o.QueryTable("user") //不知道主键就用这个过滤操作
	var user User
	err := o.QueryTable("user").Filter("Id", cid).One(&user)
	if err == nil {
		const lll = "2006-01-02"
		user.Updated = time.Now()
		switch fieldname {
		case "name":
			user.Username = value
			// 查询是否存在，做唯一性判断
			err = qs.Filter("username", value).One(&user)
			if err == orm.ErrMultiRows {
				//		// 多条的时候报错
				//		fmt.Printf("Returned Multi Rows Not One")
				return err
			} else if err == orm.ErrNoRows {
				// 没有找到记录 说明这个用户名没有重名，可以更新
				// fmt.Printf("Not row found")
				_, err = o.Update(&user, "Username", "Updated")
				return err
			}
		case "Nickname":
			// 判断是否存在相同的昵称
			err = qs.Filter("nickname", value).One(&user)
			if err != nil { // 没查到记录
				user.Nickname = value
				_, err = o.Update(&user, "Nickname", "Updated")
				return err
			} else {
				return err
			}
		case "Password":
			//这里要加密
			md5Ctx := md5.New()
			md5Ctx.Write([]byte(value))
			cipherStr := md5Ctx.Sum(nil)
			user.Password = hex.EncodeToString(cipherStr)
			_, err = o.Update(&user, "Password", "Updated")
			return err
		case "Email":
			user.Email = value
			_, err = o.Update(&user, "Email", "Updated")
			return err
		case "Sex":
			user.Sex = value
			_, err = o.Update(&user, "Sex", "Updated")
			return err
		case "IsPartyMember":
			if value == "true" {
				user.IsPartyMember = true
			} else {
				user.IsPartyMember = false
			}
			_, err = o.Update(&user, "IsPartyMember", "Updated")
			return err
		case "Department":
			user.Department = value
			_, err = o.Update(&user, "Department", "Updated") //这里不能用&user
			return err
		case "Secoffice":
			user.Secoffice = value
			_, err = o.Update(&user, "Secoffice", "Updated") //这里不能用&user
			return err
		case "Ip":
			user.Ip = value
			_, err = o.Update(&user, "Ip", "Updated") //这里不能用&user
			return err
		case "Port":
			user.Port = value
			_, err = o.Update(&user, "Port", "Updated") //这里不能用&user
			return err
		case "Status":
			//转成int
			user.Status, err = strconv.Atoi(value)
			if err != nil {
				return err
			}
			_, err = o.Update(&user, "Status", "Updated") //这里不能用&user
			return err
		case "role":
			user.Role = value
			_, err = o.Update(&user, "Role", "Updated") //这里不能用&user
			return err
		}
		// 指定多个字段
		// o.Update(&user, "Field1", "Field2", ...)这个试验没成功
	} else {
		return err
	}
	return nil
}

type UserAvatarUrl struct {
	User       `xorm:"extends"`
	UserAvatar `xorm:"extends"`
}

// 取出用户头像
func GetUserAvatorUrl(uid int64) ([]*UserAvatarUrl, error) {
	useravatarurl := make([]*UserAvatarUrl, 0)
	return useravatarurl, engine.Table("user").Join("INNER", "user_avatar", "user.id = user_avatar.uid").Where("user.id=?", uid).Desc("user_avatar.createtime").Find(&useravatarurl)
}

type UserAppreciationUrl struct {
	User             `xorm:"extends"`
	UserAppreciation `xorm:"extends"`
}

// 取出用户赞赏码
func GetUserAppreciationUrl(uid int64) ([]*UserAppreciationUrl, error) {
	userappreciationurl := make([]*UserAppreciationUrl, 0)
	return userappreciationurl, engine.Table("user").Join("INNER", "user_appreciation", "user.id = user_appreciation.uid").Where("user.id=?", uid).Desc("user_appreciation.createtime").Find(&userappreciationurl)
}

func ValidateUser(user User) error {
	cond := orm.NewCondition()
	cond1 := cond.Or("status", 1).Or("status", 2)
	cond2 := cond.AndCond(cond1).And("username", user.Username).And("password", user.Password)
	orm := orm.NewOrm()
	var u User
	qs := orm.QueryTable("user")
	qs = qs.SetCond(cond2)
	err := qs.One(&u)
	if err != nil {
		return err
	} else if u.Username == "" {
		return errors.New("用户名或密码错误！或用户被禁止！")
	} else {
		return nil
	}
}

func CheckUname(user User) error {
	orm := orm.NewOrm()
	var u User
	qs := orm.QueryTable("user")
	err := qs.Filter("Username", user.Username).One(&u)
	if err != nil {
		return err
	}
	return nil
}

func GetUname(user User) ([]*User, error) {
	orm := orm.NewOrm()
	users := make([]*User, 0)
	qs := orm.QueryTable("user")
	_, err := qs.Filter("Username__contains", user.Username).All(&users)
	if err != nil {
		return users, err
	}
	return users, err
}

func (u *User) Valid(v *validation.Validation) {
	if u.Password != u.Repassword {
		v.SetError("Repassword", "两次输入的密码不一样")
	}
}

// 验证用户信息
func checkUser(u *User) (err error) {
	valid := validation.Validation{}
	b, _ := valid.Valid(&u)
	if !b {
		for _, err := range valid.Errors {
			log.Println(err.Key, err.Message)
			return errors.New(err.Message)
		}
	}
	return nil
}

/************************************************************/
//取出所有用户
func GetUsers() (users []*User, err error) {
	o := orm.NewOrm()
	user := new(User)
	qs := o.QueryTable(user)
	// var offset int64
	// if page <= 1 {
	// 	offset = 0
	// } else {
	// 	offset = (page - 1) * page_size
	// }
	_, err = qs.All(&users)
	if err != nil {
		return nil, err
	}
	// count, _ = qs.Count()
	return users, err
}

func GetUsersPage(limit, offset int64, sort, searchText string) (users []*User, count int64, err error) {
	o := orm.NewOrm()
	user := new(User)
	qs := o.QueryTable(user)
	if searchText != "" {
		cond := orm.NewCondition()
		cond1 := cond.Or("Username__contains", searchText).Or("Nickname__contains", searchText).Or("Email__contains", searchText).Or("Sex__contains", searchText).Or("Department__contains", searchText).Or("Secoffice__contains", searchText)
		// cond2 := cond.AndCond(cond1).And("ProjectId", id)
		qs = qs.SetCond(cond1)
		_, err = qs.Limit(limit, offset).OrderBy(sort).All(&users)

		// qs.Limit(limit, offset).OrderBy(sort).All(&users)
		count, err = qs.Count()
	} else {
		qs.Limit(limit, offset).OrderBy(sort).All(&users)
		count, err = qs.Count()
	}

	return users, count, err
}

// get user list
func Getuserlist(page int64, page_size int64, sort string) (users []orm.Params, count int64) {
	o := orm.NewOrm()
	user := new(User)
	qs := o.QueryTable(user)
	var offset int64
	if page <= 1 {
		offset = 0
	} else {
		offset = (page - 1) * page_size
	}
	qs.Limit(page_size, offset).OrderBy(sort).Values(&users)
	count, _ = qs.Count()
	return users, count
}

func GetAllusers(page int64, page_size int64, sort string) (users []*User, count int64) {
	o := orm.NewOrm()
	user := new(User)
	qs := o.QueryTable(user)
	var offset int64
	if page <= 1 {
		offset = 0
	} else {
		offset = (page - 1) * page_size
	}
	qs.Limit(page_size, offset).OrderBy(sort).All(&users)
	count, _ = qs.Count()
	return users, count
}

// 根据分院和科室名称查所有用户，只有状态1的
func GetUsersbySec(department, secoffice string) (users []*User, count int, err error) {
	o := orm.NewOrm()
	// cates := make([]*Category, 0)
	qs := o.QueryTable("user")
	//这里进行过滤
	_, err = qs.Filter("Department", department).Filter("Secoffice", secoffice).Filter("Status", 1).OrderBy("Username").All(&users)
	if err != nil {
		return nil, 0, err
	}
	// _, err = qs.OrderBy("-created").All(&cates)
	// _, err := qs.All(&cates)
	count = len(users)
	return users, count, err
}

// 根据分院名称查所有用户——适用于没有科室的部门
// 查出所有人员，只有分院（部门）而没科室字段的人员，只有状态1的
func GetUsersbySecOnly(department string) (users []*User, count int, err error) {
	o := orm.NewOrm()
	// cates := make([]*Category, 0)
	qs := o.QueryTable("user")
	//这里进行过滤
	_, err = qs.Filter("Department", department).Filter("Secoffice", "").Filter("Status", 1).OrderBy("Username").All(&users)
	if err != nil {
		return nil, 0, err
	}
	// _, err = qs.OrderBy("-created").All(&cates)
	// _, err := qs.All(&cates)
	count = len(users)
	return users, count, err
}

// 根据科室id查所有用户
func GetUsersbySecId(secofficeid string) (users []*User, count int, err error) {
	o := orm.NewOrm()
	// cates := make([]*Category, 0)
	qs := o.QueryTable("user")
	//这里进行过滤
	secid, err := strconv.ParseInt(secofficeid, 10, 64)
	if err != nil {
		return nil, 0, err
	}
	//由secid查自身科室名称
	secoffice, err := GetAdminDepartbyId(secid)
	if err != nil {
		return nil, 0, err
	}
	//由secoffice的pid查分院名称
	department, err := GetAdminDepartbyId(secoffice.ParentId)
	if err != nil {
		return nil, 0, err
	}
	//由分院名称和科室名称查所有用户
	_, err = qs.Filter("Department", department.Title).Filter("Secoffice", secoffice.Title).OrderBy("Username").All(&users)
	if err != nil {
		return nil, 0, err
	}
	// _, err = qs.OrderBy("-created").All(&cates)
	// _, err := qs.All(&cates)
	count = len(users)
	return users, count, err
}

// 添加用户
func AddUser(u *User) (int64, error) {
	if err := checkUser(u); err != nil {
		return 0, err
	}
	o := orm.NewOrm()
	user := new(User)
	user.Username = u.Username
	// user.Password = Strtomd5(u.Password)
	h := md5.New()
	h.Write([]byte(u.Password))
	user.Password = hex.EncodeToString(h.Sum(nil))
	user.Nickname = u.Nickname
	user.Email = u.Email
	user.Remark = u.Remark
	user.Status = u.Status

	id, err := o.Insert(user)
	return id, err
}

//更新用户
// func UpdateUser(u *User) (int64, error) {
// 	if err := checkUser(u); err != nil {
// 		return 0, err
// 	}
// 	o := orm.NewOrm()
// 	user := make(orm.Params)
// 	if len(u.Username) > 0 {
// 		user["Username"] = u.Username
// 	}
// 	if len(u.Nickname) > 0 {
// 		user["Nickname"] = u.Nickname
// 	}
// 	if len(u.Email) > 0 {
// 		user["Email"] = u.Email
// 	}
// 	if len(u.Remark) > 0 {
// 		user["Remark"] = u.Remark
// 	}
// 	if len(u.Password) > 0 {
// 		user["Password"] = Strtomd5(u.Password)
// 	}
// 	if u.Status != 0 {
// 		user["Status"] = u.Status
// 	}
// 	if len(user) == 0 {
// 		return 0, errors.New("update field is empty")
// 	}
// 	var table User
// 	num, err := o.QueryTable(table).Filter("Id", u.Id).Update(user)
// 	return num, err
// }

//这个作废，用在线修改代替
// func UpdateUser(userid, nickname, email, password string) error {
// 	id, err := strconv.ParseInt(userid, 10, 64)
// 	o := orm.NewOrm()
// 	user := User{Id: id}
// 	user.Nickname = nickname
// 	user.Email = email
// 	if password != "" {
// 		user.Password = password
// 		_, err = o.Update(&user, "password", "nickname", "email")
// 		if err != nil {
// 			return err
// 		}
// 	} else {
// 		_, err = o.Update(&user, "nickname", "email")
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// 更新用户登陆时间
func UpdateUserlastlogintime(username string) error {
	o := orm.NewOrm()
	user := make(orm.Params)
	if len(username) > 0 {
		user["Lastlogintime"] = time.Now()
	}

	if len(username) == 0 {
		return errors.New("update field is empty")
	}
	var table User
	_, err := o.QueryTable(table).Filter("Username", username).Update(user)
	return err
}

func DelUserById(Id int64) (int64, error) {
	o := orm.NewOrm()
	status, err := o.Delete(&User{Id: Id})
	return status, err
}

// ###*****这里特别注意，这个是用户名，是汉语拼音，不是Nickname！！！！
func GetUserByUsername(username string) (user User, err error) {
	o := orm.NewOrm()
	qs := o.QueryTable("user") //不知道主键就用这个过滤操作
	//进行编号唯一性检查
	err = qs.Filter("username", username).One(&user)
	if err != nil {
		return user, err
	}
	return user, err
}

// 根据ip查询用户
func GetUserByIp(ip string) (user User, err error) {
	o := orm.NewOrm()
	// var user User
	err = o.QueryTable("user").Filter("ip", ip).One(&user)
	if err == orm.ErrMultiRows {
		// 多条的时候报错
		// fmt.Printf("Returned Multi Rows Not One")
		return user, err
	} else if err == orm.ErrNoRows {
		// 没有找到记录
		// fmt.Printf("Not row found")
		return user, err
	} else {
		return user, err
	}
}

// 根据用户nickname取得用户
func GetUserByNickname(nickname string) (user User, err error) {
	o := orm.NewOrm()
	qs := o.QueryTable("user") //不知道主键就用这个过滤操作
	//进行编号唯一性检查
	// user = User{Nickname: nickname}
	// // useropenid := UserOpenID{OpenID: openid}
	// // Three return values：Is Created，Object Id，Error
	// if created, id, err := o.ReadOrCreate(&user, "Nickname"); err == nil {
	// 	if created {
	// 		// fmt.Println("New Insert an object. Id:", id)
	// 		return user, id, err
	// 	} else {
	// 		// fmt.Println("Get an object. Id:", id)
	// 		return user, id, err
	// 	}
	// }

	err = qs.Filter("Nickname", nickname).One(&user)
	if err != nil {
		return user, err
	}
	return user, err
}

// wx根据用户nickname取得用户
func GetwxUserByNickname(nickname string) (user User, userid int64, err error) {
	o := orm.NewOrm()
	qs := o.QueryTable("user") //不知道主键就用这个过滤操作
	//进行编号唯一性检查
	user = User{Nickname: nickname}
	// useropenid := UserOpenID{OpenID: openid}
	// Three return values：Is Created，Object Id，Error
	if created, id, err := o.ReadOrCreate(&user, "Nickname"); err == nil {
		if created {
			// fmt.Println("New Insert an object. Id:", id)
			return user, id, err
		} else {
			// fmt.Println("Get an object. Id:", id)
			return user, id, err
		}
	}

	qs.Filter("Nickname", nickname).One(&user)
	return user, user.Id, err
}

// 取到一个用户数据，不是数组，所以table无法显示
func GetUserByUserId(userid int64) (user User, err error) {
	user = User{Id: userid}
	o := orm.NewOrm()
	err = o.Read(&user) //这里是默认主键查询。=(&user,"Id")
	return user, err
}

// *********初始化数据库中的用户********
func InsertUser() {
	fmt.Println("insert user ...")
	// u := new(User)
	var u User
	u.Username = "admin"
	u.Nickname = "Hotqin888"
	Pwd1 := "admin"
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(Pwd1))
	cipherStr := md5Ctx.Sum(nil)
	u.Password = hex.EncodeToString(cipherStr)
	// u.Password = Pwdhash("admin")
	u.Email = "504284@qq.com"
	u.Remark = "I'm admin"
	u.Status = 1
	u.Role = "1"
	id, err := SaveUser(u)
	// o = orm.NewOrm()
	// o.Insert(u)
	// fmt.Println("insert user end")
	if err == nil && id > 0 {
		fmt.Println("insert user end")
	} else {
		log.Println(err)
	}

	fmt.Println("insert role ...")
	// r := new(Role)
	var r Role
	r.Rolename = "admin"
	r.Rolenumber = "1"
	r.Status = "0"

	id, err = SaveRole(r)
	if err == nil && id > 0 {
		fmt.Println("insert role end")
	} else {
		log.Println(err)
		//重新获取roleid
		role, err := GetRoleByRolename("admin")
		if err != nil {
			log.Println(err)
		} else {
			id = role.Id
		}
	}

	// //
	// user_admin, err := GetUserByUsername("admin")
	// if err != nil {
	// 	log.Println(err)
	// }
	// //将用户admin加入到角色admin里

	// e.AddGroupingPolicy(strconv.FormatInt(user_admin.Id, 10), "role_"+strconv.FormatInt(id, 10))

	// err = AddUserRole(user.Id, id)
	// if err != nil {
	// 	log.Println(err)
	// }
	//添加admin角色的权限/*

	//匿名用户角色
	r.Rolename = "anonymous"
	r.Rolenumber = "5"
	r.Status = "0"
	// r.Remark = "I'm a admin role"
	// r.Status = 2
	// r.Title = "Admin role"
	// o.Insert(r)
	id, err = SaveRole(r)
	if err == nil && id > 0 {
		fmt.Println("insert role end")
	} else {
		log.Println(err)
	}

	r.Rolename = "everyone"
	r.Rolenumber = "5"
	r.Status = "0"
	// r.Remark = "I'm a admin role"
	// r.Status = 2
	// r.Title = "Admin role"
	// o.Insert(r)
	id, err = SaveRole(r)
	if err == nil && id > 0 {
		fmt.Println("insert role end")
	} else {
		log.Println(err)
	}

	r.Rolename = "isme"
	r.Rolenumber = "4"
	r.Status = "0"
	// r.Remark = "I'm a admin role"
	// r.Status = 2
	// r.Title = "Admin role"
	// o.Insert(r)
	id, err = SaveRole(r)
	if err == nil && id > 0 {
		fmt.Println("insert role end")
	} else {
		log.Println(err)
	}
	// return err
}

// func InsertGroup() {
// 	fmt.Println("insert group ...")
// 	g := new(Group)
// 	g.Name = "APP"
// 	g.Title = "System"
// 	g.Sort = 1
// 	g.Status = 2
// 	o.Insert(g)
// 	fmt.Println("insert group end")
// }

func GetRoleByUsername(username string) (roles []*Role, count int64, err error) { //*Topic, []*Attachment, error
	roles = make([]*Role, 0)
	o := orm.NewOrm()
	// role := new(Role)
	count, err = o.QueryTable("role").Filter("Users__User__Username", username).All(&roles)
	return roles, count, err
	// 通过 post title 查询这个 post 有哪些 tag
	// var tags []*Tag
	// num, err := dORM.QueryTable("tag").Filter("Posts__Post__Title", "Introduce Beego ORM").All(&tags)
}
