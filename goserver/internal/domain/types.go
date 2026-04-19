package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AttachedFile struct {
	OpenAIFileID string `bson:"openAiFileId" json:"openAiFileId"`
	OriginalName string `bson:"originalName" json:"originalName"`
	MimeType     string `bson:"mimeType" json:"mimeType"`
	Size         *int64 `bson:"size" json:"size"`
}

type Job struct {
	ObjectID           primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	ID                 string             `bson:"-" json:"id"`
	Query              string             `bson:"query" json:"query"`
	CustomID           string             `bson:"customId" json:"customId"`
	SubmissionID       string             `bson:"submissionId" json:"submissionId"`
	SubmissionIndex    int                `bson:"submissionIndex" json:"submissionIndex"`
	SubmissionSize     int                `bson:"submissionSize" json:"submissionSize"`
	SubmissionType     string             `bson:"submissionType" json:"submissionType"`
	PromptTemplate     *string            `bson:"promptTemplate" json:"promptTemplate"`
	TemplateRecord     map[string]any     `bson:"templateRecord" json:"templateRecord"`
	AttachedFiles      []AttachedFile     `bson:"attachedFiles" json:"attachedFiles"`
	Model              string             `bson:"model" json:"model"`
	ReasoningEffort    *string            `bson:"reasoningEffort" json:"reasoningEffort"`
	Status             string             `bson:"status" json:"status"`
	BatchID            *string            `bson:"batchId" json:"batchId"`
	InputFileID        *string            `bson:"inputFileId" json:"inputFileId"`
	OutputFileID       *string            `bson:"outputFileId" json:"outputFileId"`
	ErrorFileID        *string            `bson:"errorFileId" json:"errorFileId"`
	RequestCounts      map[string]any     `bson:"requestCounts" json:"requestCounts"`
	ResultText         string             `bson:"resultText" json:"resultText"`
	ResultResponseBody map[string]any     `bson:"resultResponseBody" json:"resultResponseBody"`
	LatestOutputLine   map[string]any     `bson:"latestOutputLine" json:"latestOutputLine"`
	LatestErrorLine    map[string]any     `bson:"latestErrorLine" json:"latestErrorLine"`
	LastSyncedAt       *time.Time         `bson:"lastSyncedAt" json:"lastSyncedAt"`
	CompletedAt        *time.Time         `bson:"completedAt" json:"completedAt"`
	OpenAIBatch        map[string]any     `bson:"openaiBatch" json:"openaiBatch"`
	CreatedAt          time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt          time.Time          `bson:"updatedAt" json:"updatedAt"`
}

func (job *Job) NormalizeID() {
	if job == nil {
		return
	}

	job.ID = job.ObjectID.Hex()
}

type SubmissionIteration struct {
	ObjectID         primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	ID               string             `bson:"-" json:"id"`
	JobID            string             `bson:"jobId" json:"jobId"`
	SubmissionID     string             `bson:"submissionId" json:"submissionId"`
	SubmissionType   string             `bson:"submissionType" json:"submissionType"`
	IterationNumber  int                `bson:"iterationNumber" json:"iterationNumber"`
	Kind             string             `bson:"kind" json:"kind"`
	CustomID         string             `bson:"customId" json:"customId"`
	InputText        string             `bson:"inputText" json:"inputText"`
	RequestBody      map[string]any     `bson:"requestBody" json:"requestBody"`
	PreviousResponse *string            `bson:"previousResponseId" json:"previousResponseId"`
	ResponseID       *string            `bson:"responseId" json:"responseId"`
	BatchID          *string            `bson:"batchId" json:"batchId"`
	InputFileID      *string            `bson:"inputFileId" json:"inputFileId"`
	OutputFileID     *string            `bson:"outputFileId" json:"outputFileId"`
	ErrorFileID      *string            `bson:"errorFileId" json:"errorFileId"`
	RequestCounts    map[string]any     `bson:"requestCounts" json:"requestCounts"`
	Status           string             `bson:"status" json:"status"`
	ResultText       string             `bson:"resultText" json:"resultText"`
	ResultResponse   map[string]any     `bson:"resultResponseBody" json:"resultResponseBody"`
	LatestOutputLine map[string]any     `bson:"latestOutputLine" json:"latestOutputLine"`
	LatestErrorLine  map[string]any     `bson:"latestErrorLine" json:"latestErrorLine"`
	ToolCalls        []map[string]any   `bson:"toolCalls" json:"toolCalls"`
	ToolOutputs      []map[string]any   `bson:"toolOutputs" json:"toolOutputs"`
	NextIterationID  *string            `bson:"nextIterationId" json:"nextIterationId"`
	FollowUpState    string             `bson:"followUpState" json:"followUpState"`
	FollowUpClaimed  *time.Time         `bson:"followUpClaimedAt" json:"followUpClaimedAt"`
	LastSyncedAt     *time.Time         `bson:"lastSyncedAt" json:"lastSyncedAt"`
	CompletedAt      *time.Time         `bson:"completedAt" json:"completedAt"`
	OpenAIBatch      map[string]any     `bson:"openaiBatch" json:"openaiBatch"`
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`
}

func (iteration *SubmissionIteration) NormalizeID() {
	if iteration == nil {
		return
	}

	iteration.ID = iteration.ObjectID.Hex()
}

type ProcedureStep struct {
	ID              string  `bson:"id" json:"id"`
	StepNumber      int     `bson:"stepNumber" json:"stepNumber"`
	Prompt          string  `bson:"prompt" json:"prompt"`
	Model           string  `bson:"model" json:"model"`
	ReasoningEffort *string `bson:"reasoningEffort" json:"reasoningEffort"`
}

type Procedure struct {
	ObjectID  primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	ID        string             `bson:"-" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Steps     []ProcedureStep    `bson:"steps" json:"steps"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

func (procedure *Procedure) NormalizeID() {
	if procedure == nil {
		return
	}

	procedure.ID = procedure.ObjectID.Hex()
}

type ProcedureExecutionStep struct {
	ID                 string         `bson:"id" json:"id"`
	StepNumber         int            `bson:"stepNumber" json:"stepNumber"`
	Prompt             string         `bson:"prompt" json:"prompt"`
	Model              string         `bson:"model" json:"model"`
	ReasoningEffort    *string        `bson:"reasoningEffort" json:"reasoningEffort"`
	Status             string         `bson:"status" json:"status"`
	StepInput          string         `bson:"stepInput" json:"stepInput"`
	JobID              *string        `bson:"jobId" json:"jobId"`
	BatchID            *string        `bson:"batchId" json:"batchId"`
	StartedAt          *time.Time     `bson:"startedAt" json:"startedAt"`
	CompletedAt        *time.Time     `bson:"completedAt" json:"completedAt"`
	LastSyncedAt       *time.Time     `bson:"lastSyncedAt" json:"lastSyncedAt"`
	ExecutionDuration  *int64         `bson:"executionDurationMs" json:"executionDurationMs"`
	ResultText         string         `bson:"resultText" json:"resultText"`
	ResultResponseBody map[string]any `bson:"resultResponseBody" json:"resultResponseBody"`
	LatestError        map[string]any `bson:"latestError" json:"latestError"`
}

type ProcedureExecution struct {
	ObjectID         primitive.ObjectID       `bson:"_id,omitempty" json:"-"`
	ID               string                   `bson:"-" json:"id"`
	ProcedureID      string                   `bson:"procedureId" json:"procedureId"`
	ProcedureName    string                   `bson:"procedureName" json:"procedureName"`
	InitialPrompt    string                   `bson:"initialPrompt" json:"initialPrompt"`
	Status           string                   `bson:"status" json:"status"`
	CurrentStepIndex *int                     `bson:"currentStepIndex" json:"currentStepIndex"`
	StartedAt        *time.Time               `bson:"startedAt" json:"startedAt"`
	CompletedAt      *time.Time               `bson:"completedAt" json:"completedAt"`
	LastRefreshedAt  *time.Time               `bson:"lastRefreshedAt" json:"lastRefreshedAt"`
	Steps            []ProcedureExecutionStep `bson:"steps" json:"steps"`
	CreatedAt        time.Time                `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time                `bson:"updatedAt" json:"updatedAt"`
}

func (execution *ProcedureExecution) NormalizeID() {
	if execution == nil {
		return
	}

	execution.ID = execution.ObjectID.Hex()
}
