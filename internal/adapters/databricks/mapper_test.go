package databricks

import (
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"

	"github.com/youenchene/databricks-tui/internal/domain/cluster"
	domainJob "github.com/youenchene/databricks-tui/internal/domain/job"
	"github.com/youenchene/databricks-tui/internal/domain/notebook"
)

// --- toDomainCluster ---

func TestToDomainCluster(t *testing.T) {
	d := compute.ClusterDetails{
		ClusterId:      "0123-abc-def",
		ClusterName:    "prod-etl",
		State:          compute.StateRunning,
		SparkVersion:   "14.3.x-scala2.12",
		NodeTypeId:     "Standard_DS3_v2",
		NumWorkers:     4,
		CreatorUserName: "user@company.com",
		StartTime:      1700000000000,
	}

	result := toDomainCluster(d)

	assert.Equal(t, "0123-abc-def", result.ID)
	assert.Equal(t, "prod-etl", result.Name)
	assert.Equal(t, cluster.StateRunning, result.State)
	assert.Equal(t, "14.3.x-scala2.12", result.SparkVersion)
	assert.Equal(t, "Standard_DS3_v2", result.NodeTypeID)
	assert.Equal(t, int32(4), result.NumWorkers)
	assert.Equal(t, "user@company.com", result.Creator)
	assert.False(t, result.CreatedAt.IsZero())
}

func TestToDomainCluster_Terminated(t *testing.T) {
	d := compute.ClusterDetails{
		ClusterId: "dead-cluster",
		State:     compute.StateTerminated,
		StartTime: 0,
	}

	result := toDomainCluster(d)
	assert.Equal(t, cluster.StateTerminated, result.State)
	assert.True(t, result.CreatedAt.IsZero(), "zero StartTime should yield zero CreatedAt")
}

// --- mapClusterState ---

func TestMapClusterState(t *testing.T) {
	tests := []struct {
		name  string
		sdk   compute.State
		dom   cluster.State
	}{
		{"pending", compute.StatePending, cluster.StatePending},
		{"running", compute.StateRunning, cluster.StateRunning},
		{"restarting", compute.StateRestarting, cluster.StateRestarting},
		{"resizing", compute.StateResizing, cluster.StateResizing},
		{"terminating", compute.StateTerminating, cluster.StateTerminating},
		{"terminated", compute.StateTerminated, cluster.StateTerminated},
		{"error", compute.StateError, cluster.StateError},
		{"unknown", compute.State("garbage"), cluster.StateUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.dom, mapClusterState(tt.sdk))
		})
	}
}

// --- jobSettingsName ---

func TestJobSettingsName(t *testing.T) {
	s := &jobs.JobSettings{Name: "my-job"}
	assert.Equal(t, "my-job", jobSettingsName(s))

	assert.Equal(t, "", jobSettingsName(nil))
}

// --- jobSettingsCron ---

func TestJobSettingsCron(t *testing.T) {
	s := &jobs.JobSettings{
		Schedule: &jobs.CronSchedule{QuartzCronExpression: "0 0 * * *"},
	}
	assert.Equal(t, "0 0 * * *", jobSettingsCron(s))

	assert.Equal(t, "", jobSettingsCron(nil))

	noSchedule := &jobs.JobSettings{}
	assert.Equal(t, "", jobSettingsCron(noSchedule))
}

// --- mapJobRunState ---

func TestMapJobRunState(t *testing.T) {
	tests := []struct {
		name string
		sdk  *jobs.RunState
		dom  domainJob.State
	}{
		{
			name: "nil state",
			sdk:  nil,
			dom:  domainJob.StateUnknown,
		},
		{
			name: "pending",
			sdk:  &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStatePending},
			dom:  domainJob.StatePending,
		},
		{
			name: "running",
			sdk:  &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
			dom:  domainJob.StateRunning,
		},
		{
			name: "terminated succeeded",
			sdk: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateTerminated,
				ResultState:    jobs.RunResultStateSuccess,
			},
			dom: domainJob.StateSucceeded,
		},
		{
			name: "terminated failed",
			sdk: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateTerminated,
				ResultState:    jobs.RunResultStateFailed,
			},
			dom: domainJob.StateFailed,
		},
		{
			name: "terminated canceled",
			sdk: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateTerminated,
				ResultState:    jobs.RunResultStateCanceled,
			},
			dom: domainJob.StateCanceled,
		},
		{
			name: "terminating succeeded",
			sdk: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateTerminating,
				ResultState:    jobs.RunResultStateSuccess,
			},
			dom: domainJob.StateSucceeded,
		},
		{
			name: "terminated unknown result",
			sdk: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateTerminated,
				ResultState:    jobs.RunResultState("garbage"),
			},
			dom: domainJob.StateUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.dom, mapJobRunState(tt.sdk))
		})
	}
}

// --- msToTime ---

func TestMsToTime(t *testing.T) {
	tests := []struct {
		name string
		ms   int64
		zero bool
	}{
		{"zero ms yields zero time", 0, true},
		{"valid timestamp", 1700000000000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := msToTime(tt.ms)
			if tt.zero {
				assert.True(t, result.IsZero())
			} else {
				assert.False(t, result.IsZero())
				assert.Equal(t, tt.ms, result.UnixMilli())
			}
		})
	}
}

func TestMsToTime_ValidConversion(t *testing.T) {
	result := msToTime(1700000000000)
	assert.False(t, result.IsZero())
	assert.Equal(t, int64(1700000000000), result.UnixMilli())
}

// --- mapLanguage ---

func TestMapLanguage(t *testing.T) {
	tests := []struct {
		name string
		sdk  workspace.Language
		dom  notebook.Language
	}{
		{"python", workspace.LanguagePython, notebook.LangPython},
		{"scala", workspace.LanguageScala, notebook.LangScala},
		{"sql", workspace.LanguageSql, notebook.LangSQL},
		{"r", workspace.LanguageR, notebook.LangR},
		{"unknown defaults to python", workspace.Language("JAVA"), notebook.LangPython},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.dom, mapLanguage(tt.sdk))
		})
	}
}

// --- mapSDKTask ---

func TestMapSDKTask_Notebook(t *testing.T) {
	task := jobs.Task{
		TaskKey:     "ingest",
		Description: "Ingest data",
		DependsOn:   []jobs.TaskDependency{},
		NotebookTask: &jobs.NotebookTask{
			NotebookPath: "/Shared/ingest.py",
		},
	}
	result := mapSDKTask(task)
	assert.Equal(t, "ingest", result.TaskKey)
	assert.Equal(t, "Ingest data", result.Description)
	assert.Equal(t, "/Shared/ingest.py", result.NotebookPath)
	assert.Equal(t, "Notebook", result.TaskType())
}

func TestMapSDKTask_SparkJar(t *testing.T) {
	task := jobs.Task{
		TaskKey: "compute",
		SparkJarTask: &jobs.SparkJarTask{
			MainClassName: "com.example.Main",
		},
	}
	result := mapSDKTask(task)
	assert.Equal(t, "com.example.Main", result.MainClassName)
	assert.Equal(t, "SparkJar", result.TaskType())
}

func TestMapSDKTask_DependsOn(t *testing.T) {
	task := jobs.Task{
		TaskKey: "transform",
		DependsOn: []jobs.TaskDependency{
			{TaskKey: "ingest"},
			{TaskKey: "validate"},
		},
	}
	result := mapSDKTask(task)
	assert.Len(t, result.DependsOn, 2)
	assert.Equal(t, "ingest", result.DependsOn[0])
	assert.Equal(t, "validate", result.DependsOn[1])
}

// --- mapSDKRunTask ---

func TestMapSDKRunTask(t *testing.T) {
	rt := jobs.RunTask{
		TaskKey: "ingest",
		State: &jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateSuccess,
		},
		StartTime:   1700000000000,
		EndTime:     1700000005000,
		RunDuration: 5000,
	}
	result := mapSDKRunTask(rt)
	assert.Equal(t, "ingest", result.TaskKey)
	assert.Equal(t, domainJob.StateSucceeded, result.State)
	assert.Equal(t, 5*time.Second, result.RunDuration)
}

func TestMapSDKRunTask_NilState(t *testing.T) {
	rt := jobs.RunTask{TaskKey: "orphan"}
	result := mapSDKRunTask(rt)
	assert.Equal(t, domainJob.StateUnknown, result.State)
}

// --- mapSDKRunOutput ---

func TestMapSDKRunOutput(t *testing.T) {
	o := &jobs.RunOutput{
		NotebookOutput: &jobs.NotebookOutput{Result: "42"},
		Logs:           "log content",
		Error:          "division by zero",
		ErrorTrace:     "line 12",
	}
	result := mapSDKRunOutput(o, nil)
	assert.Equal(t, "42", result.NotebookResult)
	assert.Equal(t, "log content", result.Logs)
	assert.Equal(t, "division by zero", result.ErrorMsg)
	assert.Equal(t, "line 12", result.ErrorTrace)
	assert.True(t, result.HasLogs())
	assert.True(t, result.HasError())
}

func TestMapSDKRunOutput_Nil(t *testing.T) {
	result := mapSDKRunOutput(nil, nil)
	assert.Equal(t, "output not available", result.Logs)
	assert.False(t, result.HasLogs())
}

// --- sub-field extractors ---

func TestTaskNotebookPath_Nil(t *testing.T) {
	assert.Equal(t, "", taskNotebookPath(nil))
}

func TestTaskMainClass_Nil(t *testing.T) {
	assert.Equal(t, "", taskMainClass(nil))
}

func TestRunOutputNotebook_Nil(t *testing.T) {
	o := &jobs.RunOutput{}
	assert.Equal(t, "", runOutputNotebook(o))
}

func TestRunOutputSQL_Nil(t *testing.T) {
	o := &jobs.RunOutput{}
	assert.Equal(t, "", runOutputSQL(o))
}
