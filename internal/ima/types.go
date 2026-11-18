package ima

import "encoding/json"

type NoteExtInfo struct {
	FolderID   string `json:"folder_id,omitempty"`
	FolderName string `json:"folder_name,omitempty"`
}

type NoteInfo struct {
	NoteID      string      `json:"note_id"`
	Title       string      `json:"title"`
	Summary     string      `json:"summary"`
	CreateTime  Int64       `json:"create_time"`
	ModifyTime  Int64       `json:"modify_time"`
	CoverImage  string      `json:"cover_image,omitempty"`
	NoteExtInfo NoteExtInfo `json:"note_ext_info,omitempty"`
}

type SearchNoteInfo struct {
	NoteBookInfo NoteInfo          `json:"note_book_info"`
	Highlight    map[string]string `json:"highlightInfo,omitempty"`
}

type SearchNotesData struct {
	SearchNoteInfos []SearchNoteInfo `json:"search_note_infos"`
	IsEnd           Bool             `json:"is_end"`
	TotalHitNum     Int64            `json:"total_hit_num"`
}

type ListNotesData struct {
	NoteBookList []NoteInfo `json:"note_book_list"`
	IsEnd        Bool       `json:"is_end"`
}

type NoteFolderInfo struct {
	FolderID       string `json:"folder_id"`
	Name           string `json:"name"`
	CreateTime     Int64  `json:"create_time"`
	ModifyTime     Int64  `json:"modify_time"`
	NoteNumber     Int64  `json:"note_number"`
	ParentFolderID string `json:"parent_folder_id"`
	FolderType     Int    `json:"folder_type"`
}

type ListNotebooksData struct {
	NoteFolderInfos []NoteFolderInfo `json:"note_folder_infos"`
	NextCursor      String           `json:"next_cursor"`
	IsEnd           Bool             `json:"is_end"`
	NextVersion     String           `json:"next_version"`
	NeedUpdate      Bool             `json:"need_update"`
}

type NoteContentData struct {
	Content string `json:"content"`
}

type NoteWriteData struct {
	NoteID string `json:"note_id"`
}

type KnowledgeBaseInfo struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	CoverURL             string   `json:"cover_url,omitempty"`
	Description          string   `json:"description,omitempty"`
	RecommendedQuestions []string `json:"recommended_questions,omitempty"`
}

type SearchedKnowledgeBaseInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	CoverURL string `json:"cover_url,omitempty"`
}

type AddableKnowledgeBaseInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SearchKnowledgeBasesData struct {
	InfoList   []SearchedKnowledgeBaseInfo `json:"info_list"`
	IsEnd      Bool                        `json:"is_end"`
	NextCursor String                      `json:"next_cursor"`
}

type AddableKnowledgeBasesData struct {
	KnowledgeBases []AddableKnowledgeBaseInfo `json:"addable_knowledge_base_list"`
	NextCursor     String                     `json:"next_cursor"`
	IsEnd          Bool                       `json:"is_end"`
}

type GetKnowledgeBasesData struct {
	Infos map[string]KnowledgeBaseInfo `json:"infos"`
}

type KnowledgeInfo struct {
	MediaID        string `json:"media_id"`
	Title          string `json:"title"`
	ParentFolderID string `json:"parent_folder_id,omitempty"`
	Highlight      string `json:"highlight_content,omitempty"`
	FolderID       string `json:"folder_id,omitempty"`
	Name           string `json:"name,omitempty"`
	FileNumber     Int64  `json:"file_number,omitempty"`
	FolderNumber   Int64  `json:"folder_number,omitempty"`
	IsTop          Bool   `json:"is_top,omitempty"`
	Raw            any    `json:"-"`
}

func (k *KnowledgeInfo) UnmarshalJSON(data []byte) error {
	type alias KnowledgeInfo
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*k = KnowledgeInfo(a)
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err == nil {
		k.Raw = raw
	}
	return nil
}

type FolderInfo struct {
	FolderID       string `json:"folder_id"`
	Name           string `json:"name"`
	FileNumber     Int64  `json:"file_number"`
	FolderNumber   Int64  `json:"folder_number"`
	ParentFolderID string `json:"parent_folder_id"`
	IsTop          Bool   `json:"is_top"`
}

type ListKnowledgeData struct {
	KnowledgeList []map[string]any `json:"knowledge_list"`
	IsEnd         Bool             `json:"is_end"`
	NextCursor    String           `json:"next_cursor"`
	CurrentPath   []FolderInfo     `json:"current_path"`
}

type SearchedKnowledgeInfo struct {
	MediaID          string `json:"media_id"`
	Title            string `json:"title"`
	ParentFolderID   string `json:"parent_folder_id,omitempty"`
	HighlightContent string `json:"highlight_content,omitempty"`
}

type SearchKnowledgeData struct {
	InfoList   []SearchedKnowledgeInfo `json:"info_list"`
	IsEnd      Bool                    `json:"is_end"`
	NextCursor String                  `json:"next_cursor"`
}

type DuplicateCheckResult struct {
	Name       string `json:"name"`
	IsRepeated Bool   `json:"is_repeated"`
}

type DuplicateCheckData struct {
	Results []DuplicateCheckResult `json:"results"`
}

type ImportURLData struct {
	URL     string `json:"url"`
	RetCode Int    `json:"ret_code"`
	MediaID string `json:"media_id,omitempty"`
}

type ImportURLsData struct {
	Results map[string]ImportURLData `json:"results"`
}

type URLInfo struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

type NotebookExtInfo struct {
	NotebookID string `json:"notebook_id"`
}

type MediaInfoData struct {
	MediaType       Int              `json:"media_type"`
	URLInfo         *URLInfo         `json:"url_info,omitempty"`
	NotebookExtInfo *NotebookExtInfo `json:"notebook_ext_info,omitempty"`
}

type COSCredential struct {
	Token        string `json:"token"`
	SecretID     string `json:"secret_id"`
	SecretKey    string `json:"secret_key"`
	StartTime    Int64  `json:"start_time"`
	ExpiredTime  Int64  `json:"expired_time"`
	AppID        string `json:"appid"`
	BucketName   string `json:"bucket_name"`
	Region       string `json:"region"`
	CustomDomain string `json:"custom_domain"`
	COSKey       string `json:"cos_key"`
}

type CreateMediaData struct {
	MediaID       string        `json:"media_id"`
	COSCredential COSCredential `json:"cos_credential"`
}

type AddKnowledgeData struct {
	MediaID string `json:"media_id"`
}
