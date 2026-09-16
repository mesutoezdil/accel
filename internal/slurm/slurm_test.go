package slurm

import "testing"

func TestParse(t *testing.T) {
	out := "JobId=101 JobName=train UserId=alice(1000) JobState=RUNNING NodeList=gpu[01-02] Nodes=gpu01 GRES=gpu:2(IDX:0-1)\n" +
		"JobId=102 JobName=old UserId=bob(1001) JobState=COMPLETED NodeList=gpu01\n" +
		"JobId=103 JobName=eval UserId=carol(1002) JobState=RUNNING NodeList=gpu07 Nodes=gpu07 GRES=gpu(IDX:3)\n"
	jobs := Parse(out, "gpu01")
	if len(jobs) != 1 {
		t.Fatalf("jobs %+v", jobs)
	}
	j := jobs["101"]
	if j.Name != "train" || j.User != "alice" || len(j.Devices) != 2 || j.Devices[1] != 1 {
		t.Fatalf("%+v", j)
	}
	if JobFromCgroup("0::/system.slice/slurmstepd.scope/job_101/step_0/user/task_0\n") != "101" {
		t.Fatal("cgroup job")
	}
	if JobFromCgroup("0::/user.slice\n") != "" {
		t.Fatal("no job")
	}
}
