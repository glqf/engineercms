// 接收远程请求，将指定文件夹下的pdf导入standard数据表，以及建立全文检索
package controllers

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	// "github.com/3xxx/engineercms/controllers/utils"
	// "github.com/3xxx/engineercms/models"
	"github.com/3xxx/engineercms/controllers/utils"
	"github.com/3xxx/engineercms/models"
	"github.com/PuerkitoBio/goquery"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/google/go-tika/tika"
	// "github.com/elastic/go-elasticsearch/v8"
	"context"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	// "sync"
	"time"
)

type ImportStandardController struct {
	web.Controller
}

// ScanRequest 扫描请求参数
type ScanRequest struct {
	Directory  string `json:"directory"`
	MaxWorkers int    `json:"max_workers"`
}

// ScanResponse 扫描响应
type ScanResponse struct {
	Success    bool          `json:"success"`
	Message    string        `json:"message"`
	TotalFiles int           `json:"total_files"`
	NewFiles   int           `json:"new_files"`
	Existing   int           `json:"existing_files"`
	Errors     []string      `json:"errors,omitempty"`
	Results    []*ScanResult `json:"results,omitempty"`
	Duration   string        `json:"duration"`
}

// @Title ScanPDFDirectory
// @Description 扫描指定目录中的PDF文件
// @Param   body    body    controllers.ScanRequest  true    "扫描请求参数"
// @Success 200 {object} controllers.ScanResponse
// @router /scan [post]
func (c *ImportStandardController) Scan() {
	// _, _, _, isadmin, _ := checkprodRole(c.Ctx)
	// if !isadmin {
	// 	c.Data["json"] = ScanResponse{
	// 		Success: false,
	// 		Message: fmt.Sprintf("非管理员！"),
	// 	}
	// 	c.ServeJSON()
	// 	return
	// }
	var req ScanRequest
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = ScanResponse{
			Success: false,
			Message: fmt.Sprintf("请求参数解析失败: %v", err),
		}
		c.ServeJSON()
		return
	}

	// 参数验证
	if req.Directory == "" {
		c.Data["json"] = ScanResponse{
			Success: false,
			Message: "目录路径不能为空",
		}
		c.ServeJSON()
		return
	}

	startTime := time.Now()

	// 创建扫描器并执行扫描
	scanner := NewPDFScanner(req.MaxWorkers)
	results, errors := scanner.ScanDirectory(req.Directory)

	duration := time.Since(startTime)

	// 统计结果
	var newFiles, existingFiles int
	for _, result := range results {
		if result.Exists {
			existingFiles++
		} else {
			newFiles++
		}
	}

	// 记录日志
	logs.Info("PDF扫描完成: 目录=%s, 总文件数=%d, 新增=%d, 已存在=%d, 错误数=%d, 耗时=%v",
		req.Directory, len(results), newFiles, existingFiles, len(errors), duration)

	// 构建错误信息
	errorMessages := make([]string, len(errors))
	for i, err := range errors {
		errorMessages[i] = err.Error()
	}

	c.Data["json"] = ScanResponse{
		Success:    len(errors) == 0,
		Message:    fmt.Sprintf("扫描完成，共处理 %d 个文件", len(results)),
		TotalFiles: len(results),
		NewFiles:   newFiles,
		Existing:   existingFiles,
		Errors:     errorMessages,
		Results:    results,
		Duration:   duration.String(),
	}

	c.ServeJSON()
}

// *********
// PDFScanner PDF文件扫描器
type PDFScanner struct {
	existingMap map[string]uint // 缓存已存在的文件ID和记录ID
}

// ScanResult 扫描结果
type ScanResult struct {
	PDFFile  *PDFFileInfo
	Exists   bool
	RecordID uint
	Error    error
}

// PDFFile 表示PDF文件信息
type PDFFileInfo struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	FileID    string    `gorm:"size:100;not null;uniqueIndex:idx_file_id_name" json:"file_id"`
	FileName  string    `gorm:"size:255;not null;uniqueIndex:idx_file_id_name" json:"file_name"`
	FilePath  string    `gorm:"size:500;not null" json:"file_path"`
	FileSize  int64     `gorm:"not null" json:"file_size"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// NewPDFScanner 创建新的PDF扫描器
func NewPDFScanner(maxWorkers int) *PDFScanner {
	return &PDFScanner{
		existingMap: make(map[string]uint),
	}
}

// ScanDirectory 扫描指定目录（递归子文件夹）
func (s *PDFScanner) ScanDirectory(dirPath string) ([]*ScanResult, []error) {
	// 预加载已存在的记录到缓存
	if err := s.preloadExistingRecords(); err != nil {
		return nil, []error{err}
	}

	// 收集结果
	var results []*ScanResult
	var errors []error
	// 递归扫描目录并顺序处理文件
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errors = append(errors, fmt.Errorf("访问路径 %s 失败: %v", path, err))
			return nil
		}

		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			result := s.processFile(path)
			results = append(results, result)
		}

		return nil
	})

	if err != nil {
		errors = append(errors, fmt.Errorf("扫描目录失败: %v", err))
	}

	return results, errors
}

// processFile 处理单个PDF文件
func (s *PDFScanner) processFile(filePath string) *ScanResult {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return &ScanResult{Error: fmt.Errorf("获取文件信息失败 %s: %v", filePath, err)}
	}

	// 生成文件ID（使用文件路径和名称的MD5）
	fileID := s.generateFileID(filePath, fileInfo.Name())

	// 检查是否已存在
	recordID, exists := s.existingMap[fileID]

	if exists {
		return &ScanResult{
			PDFFile: &PDFFileInfo{
				FileID:   fileID,
				FileName: fileInfo.Name(),
				FilePath: filePath,
				FileSize: fileInfo.Size(),
			},
			Exists:   true,
			RecordID: recordID,
		}
	}

	// 插入新记录
	pdfFile := &PDFFileInfo{
		FileID:    fileID,
		FileName:  fileInfo.Name(),
		FilePath:  filePath,
		FileSize:  fileInfo.Size(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// o := orm.NewOrm()
	// _, err = o.Insert(pdfFile)
	// if err != nil {
	// 	// 如果插入失败，可能是并发冲突，尝试查询
	// 	existingFile := &PDFFileInfo{FileID: fileID, FileName: fileInfo.Name()}
	// 	err = o.Read(existingFile, "FileID", "FileName")
	// 	if err == nil {
	// 		// 更新缓存
	// 		s.mu.Lock()
	// 		s.existingMap[fileID] = existingFile.ID
	// 		s.mu.Unlock()

	// 		return &ScanResult{
	// 			PDFFile:  existingFile,
	// 			Exists:   true,
	// 			RecordID: existingFile.ID,
	// 		}
	// 	}
	// 	return &ScanResult{Error: fmt.Errorf("插入记录失败 %s: %v", filePath, err)}
	// }

	// 这里插入elasticsearch
	file_name := filepath.Clean(fileInfo.Name())
	clean_file_name := strings.TrimPrefix(filepath.Join(string(filepath.Separator), file_name), string(filepath.Separator))

	fileSuffix := path.Ext(clean_file_name)
	if fileSuffix != ".DOC" && fileSuffix != ".doc" && fileSuffix != ".DOCX" && fileSuffix != ".docx" && fileSuffix != ".pdf" && fileSuffix != ".PDF" {
		return &ScanResult{Error: fmt.Errorf("文件类型错误，请上传doc或pdf %s", filePath)}
	}

	category, categoryname, fileNumber, year, fileName, _ := SplitStandardName(clean_file_name)
	var article_body string
	var standard models.Standard

	//纯英文下没有取到汉字字符，所以没有名称
	if fileName == "" {
		fileName = fileNumber
	}

	if category != "Atlas" {
		standard.Number = categoryname + " " + fileNumber + "-" + year
		standard.Title = fileName
	} else {
		standard.Number = fileNumber
		standard.Title = fileName
	}
	//这里增加Category
	standard.Category = categoryname //2016-7-16这里改为GBT这种，空格前的名字
	standard.Created = time.Now()
	standard.Updated = time.Now()
	// standard.Uid = uid
	standard.Route = "/attachment/standard/" + category + "/" + clean_file_name

	sid, err := models.SaveStandard(standard)
	logs.Info(sid)
	if err != nil {
		logs.Error(err)
		// 如果插入失败，可能是并发冲突，尝试查询
		// existingFile := &PDFFileInfo{FileID: fileID, FileName: fileInfo.Name()}
		// err = o.Read(existingFile, "FileID", "FileName")
		// if err == nil {
		// 	// 更新缓存
		// 	s.mu.Lock()
		// 	s.existingMap[fileID] = existingFile.ID
		// 	s.mu.Unlock()

		// 	return &ScanResult{
		// 		PDFFile:  existingFile,
		// 		Exists:   true,
		// 		RecordID: existingFile.ID,
		// 	}
		// }
		return &ScanResult{Error: fmt.Errorf("插入记录失败 %s: %v", filePath, err)}
	}

	cwd, _ := os.Getwd()

	url_path := strings.Replace(filePath, "./attachment/", "/attachment/", -1)
	f, err := os.Open(url_path)
	// logs.Info(url_path)
	if err != nil {
		logs.Error(err)
	}
	defer f.Close()

	client := tika.NewClient(nil, "http://localhost:9998")
	body, err := client.Parse(context.Background(), f)
	// body, err := client.Detect(context.Background(), f) //application/pdf
	// fmt.Println(err)
	// fmt.Println(body)

	dom, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		logs.Error(err)
	}

	dom.Find("p").Each(func(i int, selection *goquery.Selection) {
		text := strings.Replace(selection.Text(), "\n", "", -1)
		text = strings.Replace(text, " ", "", -1)
		article_body = article_body + text //selection.Text()
	})

	now := time.Now()
	year_2, month, day := now.Date()
	today_str := fmt.Sprintf("%04d-%02d-%02d", year_2, month, day)
	rand.Seed(time.Now().Unix())
	// 提取pdf第一页作为封面
	datapath := cwd + "/static/pdf/mutool.exe"

	onameexp := ".png"
	// mutool convert -O width=200 -F png -o output.png 01.pdf 1
	arg := []string{"convert", "-O", "width=300", "-F", "png", "-o", cwd + "/static/images/" + fileName + onameexp, url_path, "1"}
	logs.Info("------------", arg)
	cmd := exec.Command(datapath, arg...)
	//记录开始时间
	// start := time.Now()

	err = cmd.Start()
	if err != nil {
		fmt.Printf("err: %v", err)
	}

	doc := &Document{
		//是productid还是attachmentid
		ID: strconv.FormatInt(sid, 10), // 将int64整数转换为字符串
		// ImageURL:  "/static/images/" + strconv.Itoa(rand.Intn(8)) + "s.jpg", //1s.jpg
		ImageURL:  "/static/images/" + fileName + "1" + onameexp,
		Published: today_str, //fmt.Sprintf("%04d-%02d-%02d", jYear, jMonth, jDay),
		Title:     categoryname + " " + fileNumber + "-" + year + fileName,
		Body:      article_body,
	}

	err = Createitem(indexName, doc) //这个indexName是全局变量
	if err != nil {
		utils.FileLogs.Info(indexName + ":" + fileName + ":" + err.Error())
		return &ScanResult{Error: fmt.Errorf("加入全文检索elastic错误！ %s: %v", filePath, err)}
	}
	utils.FileLogs.Info(indexName + ":" + filePath + ":ok成功！")
	// c.Data["json"] = map[string]interface{}{"info": "ERROR", "data": "加入全文检索elastic错误！"}
	// c.ServeJSON()

	// 更新缓存
	s.existingMap[fileID] = pdfFile.ID

	return &ScanResult{
		PDFFile:  pdfFile,
		Exists:   false,
		RecordID: pdfFile.ID,
	}
}

// generateFileID 生成文件唯一ID
func (s *PDFScanner) generateFileID(filePath, fileName string) string {
	h := md5.New()
	h.Write([]byte(filePath + "::" + fileName))
	return hex.EncodeToString(h.Sum(nil))
}

// preloadExistingRecords 预加载已存在的记录
func (s *PDFScanner) preloadExistingRecords() error {
	// o := orm.NewOrm()
	// var existingFiles []*PDFFileInfo
	// _, err := o.QueryTable(new(PDFFileInfo)).All(&existingFiles)
	// if err != nil {
	// 	return err
	// }

	// s.mu.Lock()
	// defer s.mu.Unlock()

	// for _, file := range existingFiles {
	// 	s.existingMap[file.FileID] = file.ID
	// }

	return nil
}

// GetFileInfo 获取单个文件信息（如果存在返回记录ID）
func (s *PDFScanner) GetFileInfo(filePath string) (uint, bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, false, err
	}

	fileID := s.generateFileID(filePath, fileInfo.Name())

	// 检查缓存
	recordID, exists := s.existingMap[fileID]

	if exists {
		return recordID, true, nil
	}

	// 查询数据库
	o := orm.NewOrm()
	pdfFile := &PDFFileInfo{FileID: fileID, FileName: fileInfo.Name()}
	err = o.Read(pdfFile, "FileID", "FileName")
	if err == nil {
		// 更新缓存
		s.existingMap[fileID] = pdfFile.ID
		return pdfFile.ID, true, nil
	}

	return 0, false, nil
}
