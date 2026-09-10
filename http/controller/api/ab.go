package api

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/slxar/rustdesk-api-goravel/v3/global"
	requstform "github.com/slxar/rustdesk-api-goravel/v3/http/request/api"
	"github.com/slxar/rustdesk-api-goravel/v3/http/response"
	"github.com/slxar/rustdesk-api-goravel/v3/http/response/api"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
	"github.com/slxar/rustdesk-api-goravel/v3/utils"
	"gorm.io/gorm"
	"math"
	"net/http"
	"strconv"
	"strings"
)

type Ab struct {
}

// Ab
// @Tags 地址
// @Summary 地址列表
// @Description 地址列表
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.ErrorResponse
// @Router /ab [get]
// @Security BearerAuth
func (a *Ab) Ab(c *gin.Context) {
	user := service.AllService.UserService.CurUser(c)

	al := service.AllService.AddressBookService.ListByUserIdAndCollectionId(user.Id, 0, 1, 1000)
	tags := service.AllService.TagService.ListByUserIdAndCollectionId(user.Id, 0)

	tagColors := map[string]uint{}
	//将tags中的name转成一个以逗号分割的字符串
	var tagNames []string
	for _, tag := range tags.Tags {
		tagNames = append(tagNames, tag.Name)
		tagColors[tag.Name] = tag.Color
	}
	tgc, _ := json.Marshal(tagColors)
	res := &api.AbList{
		Peers:     al.AddressBooks,
		Tags:      tagNames,
		TagColors: string(tgc),
	}
	data, _ := json.Marshal(res)
	c.JSON(http.StatusOK, gin.H{
		"data": string(data),
		//"licensed_devices": 999,
	})
}

// UpAb
// @Tags 地址
// @Summary 地址更新
// @Description 地址更新
// @Accept  json
// @Produce  json
// @Param body body requstform.AddressBookForm true "地址表单"
// @Success 200 {string} string "null"
// @Failure 500 {object} response.ErrorResponse
// @Router /ab [post]
// @Security BearerAuth
func (a *Ab) UpAb(c *gin.Context) {
	abf := &requstform.AddressBookForm{}
	err := c.ShouldBindJSON(&abf)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	abd := &requstform.AddressBookFormData{}
	err = json.Unmarshal([]byte(abf.Data), abd)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	tc := map[string]uint{}
	err = json.Unmarshal([]byte(abd.TagColors), &tc)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	user := service.AllService.UserService.CurUser(c)

	err = service.AllService.AddressBookService.UpdateAddressBook(abd.Peers, user.Id)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}

	service.AllService.TagService.UpdateTags(user.Id, tc)

	c.JSON(http.StatusOK, nil)
}

// PTags
// @Tags 地址[Personal]
// @Summary 标签
// @Description 标签
// @Accept  json
// @Produce  json
// @Param guid path string true "guid"
// @Success 200 {object} model.TagList
// @Failure 500 {object} response.ErrorResponse
// @Router /ab/tags/{guid} [get]
// @Router /ab/tags/{guid} [post]
// @Security BearerAuth
func (a *Ab) PTags(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	guid := c.Param("guid")
	_, uid, cid, err := a.CheckGuid(u, guid)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	//check privileges
	if !service.AllService.AddressBookService.CheckUserReadPrivilege(u, uid, cid) {
		response.Error(c, response.TranslateMsg(c, "NoAccess"))
		return
	}
	tags := service.AllService.TagService.ListByUserIdAndCollectionId(uid, cid)
	c.JSON(http.StatusOK, tags.Tags)
}

// TagAdd
// @Tags 地址[Personal]
// @Summary 标签添加
// @Description 标签
// @Accept  json
// @Produce  json
// @Param guid path string true "guid"
// @Success 200 {string} string
// @Failure 500 {object} response.ErrorResponse
// @Router /ab/tag/add/{guid} [post]
// @Security BearerAuth
func (a *Ab) TagAdd(c *gin.Context) {

	t := &model.Tag{}
	err := c.ShouldBindJSON(t)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	u := service.AllService.UserService.CurUser(c)
	guid := c.Param("guid")
	_, uid, cid, err := a.CheckGuid(u, guid)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	//check privileges
	if !service.AllService.AddressBookService.CheckUserWritePrivilege(u, uid, cid) {
		response.Error(c, response.TranslateMsg(c, "NoAccess"))
		return
	}

	tag := service.AllService.TagService.InfoByUserIdAndNameAndCollectionId(uid, t.Name, cid)
	if tag != nil && tag.Id != 0 {
		response.Error(c, response.TranslateMsg(c, "ItemExists"))
		return
	}
	t.UserId = uid
	t.CollectionId = cid
	err = service.AllService.TagService.Create(t)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	c.String(http.StatusOK, "")
}

// TagRename
// @Tags 地址[Personal]
// @Summary 标签重命名
// @Description 标签
// @Accept  json
// @Produce  json
// @Param guid path string true "guid"
// @Success 200 {string} string
// @Failure 500 {object} response.ErrorResponse
// @Router /ab/tag/rename/{guid} [put]
// @Security BearerAuth
func (a *Ab) TagRename(c *gin.Context) {

	t := &requstform.TagRenameForm{}
	err := c.ShouldBindJSON(t)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	u := service.AllService.UserService.CurUser(c)
	guid := c.Param("guid")
	_, uid, cid, err := a.CheckGuid(u, guid)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	//check privileges
	if !service.AllService.AddressBookService.CheckUserWritePrivilege(u, uid, cid) {
		response.Error(c, response.TranslateMsg(c, "NoAccess"))
		return
	}

	tag := service.AllService.TagService.InfoByUserIdAndNameAndCollectionId(uid, t.Old, cid)
	if tag == nil || tag.Id == 0 {
		response.Error(c, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	ntag := service.AllService.TagService.InfoByUserIdAndNameAndCollectionId(uid, t.New, cid)
	if ntag != nil && ntag.Id != 0 {
		response.Error(c, response.TranslateMsg(c, "ItemExists"))
		return
	}
	tag.Name = t.New
	err = service.AllService.TagService.Update(tag)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	c.String(http.StatusOK, "")
}

// TagUpdate
// @Tags 地址[Personal]
// @Summary 标签修改颜色
// @Description 标签
// @Accept  json
// @Produce  json
// @Param guid path string true "guid"
// @Success 200 {string} string
// @Failure 500 {object} response.ErrorResponse
// @Router /ab/tag/update/{guid} [put]
// @Security BearerAuth
func (a *Ab) TagUpdate(c *gin.Context) {
	t := &requstform.TagColorForm{}
	err := c.ShouldBindJSON(t)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	u := service.AllService.UserService.CurUser(c)
	guid := c.Param("guid")
	_, uid, cid, err := a.CheckGuid(u, guid)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	//check privileges
	if !service.AllService.AddressBookService.CheckUserWritePrivilege(u, uid, cid) {
		response.Error(c, response.TranslateMsg(c, "NoAccess"))
		return
	}

	tag := service.AllService.TagService.InfoByUserIdAndNameAndCollectionId(uid, t.Name, cid)
	if tag == nil || tag.Id == 0 {
		response.Error(c, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	tag.Color = t.Color
	err = service.AllService.TagService.Update(tag)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	c.String(http.StatusOK, "")
}

// TagDel
// @Tags 地址[Personal]
// @Summary 标签删除
// @Description 标签
// @Accept  json
// @Produce  json
// @Param guid path string true "guid"
// @Success 200 {string} string
// @Failure 500 {object} response.ErrorResponse
// @Router /ab/tag/{guid} [delete]
// @Security BearerAuth
func (a *Ab) TagDel(c *gin.Context) {

	t := &[]string{}
	err := c.ShouldBind(t)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	//fmt.Println(t)
	u := service.AllService.UserService.CurUser(c)
	guid := c.Param("guid")
	_, uid, cid, err := a.CheckGuid(u, guid)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	//check privileges
	if !service.AllService.AddressBookService.CheckUserFullControlPrivilege(u, uid, cid) {
		response.Error(c, response.TranslateMsg(c, "NoAccess"))
		return
	}

	for _, name := range *t {
		tag := service.AllService.TagService.InfoByUserIdAndNameAndCollectionId(uid, name, cid)
		if tag == nil || tag.Id == 0 {
			response.Error(c, response.TranslateMsg(c, "ItemNotFound"))
			return
		}
		err = service.AllService.TagService.Delete(tag)
		if err != nil {
			response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
			return
		}
	}
	c.String(http.StatusOK, "")
}

// Personal
// @Tags 地址[Personal]
// @Summary 个人地址
// @Description 个人地址
// @Accept  json
// @Produce  json
// @Param string body string false  "string valid"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ab/personal [get]
// @Router /ab/personal [post]
// @Security BearerAuth
func (a *Ab) Personal(c *gin.Context) {
	user := service.AllService.UserService.CurUser(c)
	/**
	guid = json['guid'] ?? '',
	       name = json['name'] ?? '',
	       owner = json['owner'] ?? '',
	       note = json['note'] ?? '',
	       rule = json['rule'] ?? 0;
	*/
	if global.Config.Rustdesk.Personal == 1 {
		guid := a.ComposeGuid(user.GroupId, user.Id, 0)
		//如果返回了guid，后面的请求会有变化
		c.JSON(http.StatusOK, gin.H{
			"guid": guid,
			"name": user.Username,
			"rule": 3,
		})
	} else {
		c.JSON(http.StatusOK, nil)
	}

}

// Settings
// @Tags 地址[Personal]
// @Summary 设置
// @Description 设置
// @Accept  json
// @Produce  json
// @Param string body string false  "string valid"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ab/settings [post]
// @Security BearerAuth
func (a *Ab) Settings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"max_peer_one_ab": 0, //最大peer数，0表示不限制
	})
}

// SharedProfiles
// @Tags 地址[Personal]
// @Summary 共享地址簿
// @Description 共享
// @Accept  json
// @Produce  json
// @Param name query string false "Exact address-book name"
// @Param current query int false "页码"
// @Param pageSize query int false "每页数量"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ab/shared/profiles [get]
// @Router /ab/shared/profiles [post]
// @Security BearerAuth
func (a *Ab) SharedProfiles(c *gin.Context) {
	q, ok := bindAddressBookListQuery(c)
	if !ok {
		return
	}
	user := service.AllService.UserService.CurUser(c)
	rules := service.AllService.AddressBookService.CollectionReadRules(user)
	maxRules := make(map[uint]int)
	for _, rule := range rules {
		if rule.Rule > maxRules[rule.CollectionId] {
			maxRules[rule.CollectionId] = rule.Rule
		}
	}
	collections := service.AllService.AddressBookService.ListCollection(q.Page, q.PageSize, func(tx *gorm.DB) {
		tx.Where("(user_id = ? OR id IN ?)", user.Id, utils.Keys(maxRules))
		tx.Where("user_id IN (?)", service.DB.Model(&model.User{}).Select("id"))
		if q.Name != "" {
			tx.Where("name = ?", q.Name)
		}
		tx.Order("id ASC")
	})
	ownerIDs := make(map[uint]bool)
	for _, collection := range collections.AddressBookCollection {
		ownerIDs[collection.UserId] = true
	}
	owners := make(map[uint]*model.User)
	for _, owner := range service.AllService.UserService.ListByIds(utils.Keys(ownerIDs)) {
		owners[owner.Id] = owner
	}
	res := make([]*api.SharedProfilesPayload, 0, len(collections.AddressBookCollection))
	for _, collection := range collections.AddressBookCollection {
		owner := owners[collection.UserId]
		if owner == nil { // The owner may have been deleted after the list query.
			continue
		}
		rule := maxRules[collection.Id]
		if owner.Id == user.Id {
			rule = model.ShareAddressBookRuleRuleFullControl
		}
		res = append(res, &api.SharedProfilesPayload{
			Guid: a.ComposeGuid(owner.GroupId, owner.Id, collection.Id),
			Name: collection.Name, Owner: owner.Username, Rule: rule,
		})
	}
	c.JSON(http.StatusOK, gin.H{"total": collections.Total, "data": res})
}

// Both official management scripts and native clients use current/pageSize.
// Keep the former peer-list default of 1,000 when pagination is omitted.
func bindAddressBookListQuery(c *gin.Context) (*addressBookListQuery, bool) {
	q := &addressBookListQuery{}
	if err := c.ShouldBindQuery(q); err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError"))
		return nil, false
	}
	q.Page = q.PageNumber()
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 1000
	}
	if q.PageSize > 1000 || q.Page-1 > uint(math.MaxInt)/q.PageSize {
		response.Error(c, response.TranslateMsg(c, "ParamsError"))
		return nil, false
	}
	return q, true
}

type addressBookListQuery struct {
	requstform.PeerListQuery
	Name  string `form:"name"`
	ID    string `form:"id"`
	Alias string `form:"alias"`
}

// ParseGuid
func (a *Ab) ParseGuid(guid string) (gid, uid, cid uint) {
	//用-切割 guid
	guids := strings.Split(guid, "-")
	if len(guids) < 2 {
		return 0, 0, 0
	}
	if len(guids) != 3 {
		cid = 0
	} else {
		s, err := strconv.Atoi(guids[2])
		if err != nil {
			return 0, 0, 0
		}
		cid = uint(s)
	}
	g, err := strconv.Atoi(guids[0])
	if err != nil {
		return 0, 0, 0
	}
	gid = uint(g)
	u, err := strconv.Atoi(guids[1])
	if err != nil {
		return 0, 0, 0
	}
	uid = uint(u)
	return
}

// ComposeGuid
func (a *Ab) ComposeGuid(gid, uid, cid uint) string {
	return strconv.Itoa(int(gid)) + "-" + strconv.Itoa(int(uid)) + "-" + strconv.Itoa(int(cid))
}

// CheckGuid
func (a *Ab) CheckGuid(cu *model.User, guid string) (gid, uid, cid uint, err error) {
	gid, uid, cid = a.ParseGuid(guid)
	err = nil
	if gid == 0 || uid == 0 {
		err = errors.New("ParamsError")
		return
	}
	u := &model.User{}
	if cu.Id == uid {
		u = cu
	} else {
		u = service.AllService.UserService.InfoById(uid)
	}
	if u == nil || u.Id == 0 {
		err = errors.New("ParamsError")
		return
	}
	if u.GroupId != gid {
		err = errors.New("ParamsError")
		return
	}
	if cid == 0 && cu.Id != uid {
		err = errors.New("ParamsError")
		return
	}
	if cid > 0 {
		c := service.AllService.AddressBookService.CollectionInfoById(cid)
		if c == nil || c.Id == 0 {
			err = errors.New("ParamsError")
			return
		}
		if c.UserId != uid {
			err = errors.New("ParamsError")
			return
		}
	}
	return
}

// Peers
// @Tags 地址[Personal]
// @Summary 地址列表
// @Description 地址
// @Accept  json
// @Produce  json
// @Param current query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param ab query string false "guid"
// @Param id query string false "Device ID pattern (SQL LIKE wildcards)"
// @Param alias query string false "Alias pattern (SQL LIKE wildcards)"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ab/peers [get]
// @Router /ab/peers [post]
// @Security BearerAuth
func (a *Ab) Peers(c *gin.Context) {
	q, ok := bindAddressBookListQuery(c)
	if !ok {
		return
	}
	u := service.AllService.UserService.CurUser(c)
	guid := c.Query("ab")
	_, uid, cid, err := a.CheckGuid(u, guid)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	//check privileges
	if !service.AllService.AddressBookService.CheckUserReadPrivilege(u, uid, cid) {
		response.Error(c, response.TranslateMsg(c, "NoAccess"))
		return
	}

	al := service.AllService.AddressBookService.List(q.Page, q.PageSize, func(tx *gorm.DB) {
		tx.Where("user_id = ? AND collection_id = ?", uid, cid)
		if q.ID != "" {
			tx.Where("id LIKE ?", q.ID)
		}
		if q.Alias != "" {
			tx.Where("alias LIKE ?", q.Alias)
		}
		tx.Order("row_id ASC")
	})
	if al.AddressBooks == nil {
		al.AddressBooks = []*model.AddressBook{}
	}
	c.JSON(http.StatusOK, gin.H{
		"total":            al.Total,
		"data":             al.AddressBooks,
		"licensed_devices": 99999,
	})
}

// PeerAdd
// @Tags 地址[Personal]
// @Summary 添加地址
// @Description 添加地址
// @Accept  json
// @Produce  json
// @Param guid path string true "guid"
// @Success 200 {string} string
// @Failure 500 {object} response.ErrorResponse
// @Router /ab/peer/add/{guid} [post]
// @Security BearerAuth
func (a *Ab) PeerAdd(c *gin.Context) {
	// forceAlwaysRelay永远是字符串"false"
	//f := &gin.H{}
	f := &requstform.PersonalAddressBookForm{}
	err := c.ShouldBindJSON(f)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	u := service.AllService.UserService.CurUser(c)
	guid := c.Param("guid")
	_, uid, cid, err := a.CheckGuid(u, guid)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	//check privileges
	if !service.AllService.AddressBookService.CheckUserWritePrivilege(u, uid, cid) {
		response.Error(c, response.TranslateMsg(c, "NoAccess"))
		return
	}

	//fmt.Println(f)
	f.UserId = uid
	ab := f.ToAddressBook()
	ab.CollectionId = cid
	if ab.Platform == "" || ab.Username == "" || ab.Hostname == "" {
		peer := service.AllService.PeerService.FindById(ab.Id)
		if peer.RowId != 0 {
			ab.Platform = service.AllService.AddressBookService.PlatformFromOs(peer.Os)
			ab.Username = peer.Username
			ab.Hostname = peer.Hostname
		}
	}

	err = service.AllService.AddressBookService.AddAddressBook(ab)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	c.String(http.StatusOK, "")
}

// PeerDel
// @Tags 地址[Personal]
// @Summary 删除地址
// @Description 删除地址
// @Accept  json
// @Produce  json
// @Param guid path string true "guid"
// @Success 200 {string} string
// @Failure 500 {object} response.ErrorResponse
// @Router /ab/peer/add/{guid} [delete]
// @Security BearerAuth
func (a *Ab) PeerDel(c *gin.Context) {
	f := &[]string{}
	err := c.ShouldBind(f)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	u := service.AllService.UserService.CurUser(c)
	guid := c.Param("guid")
	_, uid, cid, err := a.CheckGuid(u, guid)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	//check privileges
	if !service.AllService.AddressBookService.CheckUserFullControlPrivilege(u, uid, cid) {
		response.Error(c, response.TranslateMsg(c, "NoAccess"))
		return
	}

	for _, id := range *f {
		ab := service.AllService.AddressBookService.InfoByUserIdAndIdAndCid(uid, id, cid)
		if ab == nil || ab.RowId == 0 {
			response.Error(c, response.TranslateMsg(c, "ItemNotFound"))
			return
		}
		err = service.AllService.AddressBookService.Delete(ab)
		if err != nil {
			response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
			return
		}
	}

	c.String(http.StatusOK, "")
}

// PeerUpdate
// @Tags 地址[Personal]
// @Summary 更新地址
// @Description 更新地址
// @Accept  json
// @Produce  json
// @Param guid path string true "guid"
// @Success 200 {string} string
// @Failure 500 {object} response.ErrorResponse
// @Router /ab/peer/update/{guid} [put]
// @Security BearerAuth
func (a *Ab) PeerUpdate(c *gin.Context) {
	f := gin.H{}
	//f := &requstform.PersonalAddressBookForm{}
	err := c.ShouldBindJSON(&f)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	u := service.AllService.UserService.CurUser(c)
	guid := c.Param("guid")
	_, uid, cid, err := a.CheckGuid(u, guid)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	//check privileges
	if !service.AllService.AddressBookService.CheckUserWritePrivilege(u, uid, cid) {
		response.Error(c, response.TranslateMsg(c, "NoAccess"))
		return
	}
	//fmt.Println(f)
	//判断f["Id"]是否存在
	fid, ok := f["id"]
	if !ok {
		response.Error(c, response.TranslateMsg(c, "ParamsError"))
		return
	}
	fidstr := fid.(string)

	ab := service.AllService.AddressBookService.InfoByUserIdAndIdAndCid(uid, fidstr, cid)
	if ab == nil || ab.RowId == 0 {
		response.Error(c, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	//允许的字段
	allowUp := []string{"password", "hash", "tags", "alias"}
	//f中的字段如果不在allowUp中，就删除
	for k := range f {
		if !utils.InArray(k, allowUp) {
			delete(f, k)
		}
	}
	//fmt.Println(f)
	if tags, _ok := f["tags"]; _ok {
		f["tags"], _ = json.Marshal(tags)
	}
	err = service.AllService.AddressBookService.UpdateByMap(ab, f)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	c.String(http.StatusOK, "")
}
