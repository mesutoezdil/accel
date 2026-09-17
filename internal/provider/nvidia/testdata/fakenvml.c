/*
 * A stand-in for `libnvidia-ml.so.1` with the exact C signatures siltide binds,
 * so the purego bindings and struct layouts can be exercised on any Linux
 * box without a GPU. Build: `cc -shared -fPIC -o libnvidia-ml.so.1 fakenvml.c`
 *
 * It reports 2 devices. Device 0 is healthy; device 1 runs on a narrow
 * PCIe link with a thermal throttle and an uncorrected ECC (error-correcting
 * code) error, so the derived health and alerts have something to chew on.
 * Xid 79 is raised on the first event wait.
 */
#include <stdint.h>
#include <string.h>
#include <unistd.h>

typedef int nvmlReturn_t;
typedef unsigned int uint;
#define OK 0
#define INSUFFICIENT 7
#define NOT_SUPPORTED 3

typedef struct { uint64_t total, free, used; } nvmlMemory_t;
typedef struct { uint gpu, memory; } nvmlUtilization_t;
typedef struct { char busIdLegacy[16]; uint domain, bus, device, pciDeviceId, pciSubSystemId; char busId[32]; } nvmlPciInfo_t;
typedef struct { uint pid; uint64_t usedGpuMemory; uint gpuInstanceId, computeInstanceId; } nvmlProcessInfo_t;
typedef struct { uint pid; uint64_t timeStamp; uint smUtil, memUtil, encUtil, decUtil; } nvmlProcessUtilizationSample_t;
typedef struct { uint64_t referenceTime, violationTime; } nvmlViolationTime_t;
typedef struct { void *device; uint64_t eventType, eventData; uint gpuInstanceId, computeInstanceId; } nvmlEventData_t;
typedef struct { uint fieldId, scopeId; int64_t timestamp, latencyUsec; uint valueType; nvmlReturn_t nvmlReturn; uint64_t value; } nvmlFieldValue_t;

static int devs[2] = {0, 1};
static int events_sent = 0;
#define IDX(d) (*(int *)(d))

nvmlReturn_t nvmlInit_v2(void) { return OK; }
nvmlReturn_t nvmlShutdown(void) { return OK; }
nvmlReturn_t nvmlDeviceGetCount_v2(uint *n) { *n = 2; return OK; }
nvmlReturn_t nvmlDeviceGetHandleByIndex_v2(uint i, void **h) { if (i > 1) return 2; *h = &devs[i]; return OK; }
nvmlReturn_t nvmlDeviceGetName(void *d, char *buf, uint n) { strncpy(buf, IDX(d) == 0 ? "Fake H100 80GB" : "Fake H100 80GB", n); return OK; }
nvmlReturn_t nvmlDeviceGetUUID(void *d, char *buf, uint n) { strncpy(buf, IDX(d) == 0 ? "GPU-fake-0000" : "GPU-fake-0001", n); return OK; }
nvmlReturn_t nvmlDeviceGetPciInfo_v3(void *d, nvmlPciInfo_t *p) { memset(p, 0, sizeof *p); strcpy(p->busId, IDX(d) == 0 ? "00000000:0A:00.0" : "00000000:0B:00.0"); return OK; }
nvmlReturn_t nvmlDeviceGetUtilizationRates(void *d, nvmlUtilization_t *u) { u->gpu = IDX(d) == 0 ? 97 : 41; u->memory = 60; return OK; }
nvmlReturn_t nvmlDeviceGetMemoryInfo(void *d, nvmlMemory_t *m) { m->total = 80ULL << 30; m->used = (IDX(d) == 0 ? 60ULL : 20ULL) << 30; m->free = m->total - m->used; return OK; }
nvmlReturn_t nvmlDeviceGetTemperature(void *d, uint sensor, uint *t) { (void)sensor; *t = IDX(d) == 0 ? 68 : 91; return OK; }
nvmlReturn_t nvmlDeviceGetPowerUsage(void *d, uint *mw) { *mw = IDX(d) == 0 ? 640000 : 300000; return OK; }
nvmlReturn_t nvmlDeviceGetEnforcedPowerLimit(void *d, uint *mw) { (void)d; *mw = 700000; return OK; }
nvmlReturn_t nvmlDeviceGetClockInfo(void *d, uint type, uint *mhz) { (void)d; *mhz = type == 0 ? 1980 : 2619; return OK; }
nvmlReturn_t nvmlDeviceGetFanSpeed(void *d, uint *pct) { (void)d; (void)pct; return NOT_SUPPORTED; }
nvmlReturn_t nvmlDeviceGetCurrentClocksEventReasons(void *d, uint64_t *r) { *r = IDX(d) == 0 ? 0 : 0x20; return OK; }
nvmlReturn_t nvmlDeviceGetTotalEccErrors(void *d, uint type, uint counter, uint64_t *n) { (void)counter; *n = type == 1 && IDX(d) == 1 ? 1 : (type == 0 ? 3 : 0); return OK; }
nvmlReturn_t nvmlDeviceGetCurrPcieLinkGeneration(void *d, uint *g) { (void)d; *g = 5; return OK; }
nvmlReturn_t nvmlDeviceGetCurrPcieLinkWidth(void *d, uint *w) { *w = IDX(d) == 0 ? 16 : 8; return OK; }
nvmlReturn_t nvmlDeviceGetMaxPcieLinkGeneration(void *d, uint *g) { (void)d; *g = 5; return OK; }
nvmlReturn_t nvmlDeviceGetMaxPcieLinkWidth(void *d, uint *w) { (void)d; *w = 16; return OK; }
nvmlReturn_t nvmlDeviceGetPcieThroughput(void *d, uint counter, uint *kbs) { (void)d; *kbs = counter == 0 ? 4000000 : 12000000; return OK; }
nvmlReturn_t nvmlDeviceGetNvLinkState(void *d, uint link, uint *on) { if (link >= 18) return 2; *on = (IDX(d) == 1 && link == 17) ? 0 : 1; return OK; }
nvmlReturn_t nvmlDeviceGetNvLinkVersion(void *d, uint link, uint *v) { (void)d; (void)link; *v = 4; return OK; }
nvmlReturn_t nvmlDeviceGetNvLinkRemotePciInfo_v2(void *d, uint link, nvmlPciInfo_t *p) { (void)link; memset(p, 0, sizeof *p); strcpy(p->busId, IDX(d) == 0 ? "00000000:0B:00.0" : "00000000:0A:00.0"); return OK; }
nvmlReturn_t nvmlDeviceGetNvLinkErrorCounter(void *d, uint link, uint counter, uint64_t *n) { *n = (IDX(d) == 1 && link == 3 && counter == 0) ? 12 : 0; return OK; }
nvmlReturn_t nvmlDeviceGetEncoderUtilization(void *d, uint *u, uint *s) { (void)d; *u = 0; *s = 1000; return OK; }
nvmlReturn_t nvmlDeviceGetDecoderUtilization(void *d, uint *u, uint *s) { (void)d; *u = 0; *s = 1000; return OK; }
nvmlReturn_t nvmlDeviceGetMigMode(void *d, uint *cur, uint *pending) { (void)d; *cur = 0; *pending = 0; return OK; }
nvmlReturn_t nvmlDeviceGetMaxMigDeviceCount(void *d, uint *n) { (void)d; *n = 0; return OK; }
nvmlReturn_t nvmlDeviceGetMigDeviceHandleByIndex(void *d, uint i, void **h) { (void)d; (void)i; (void)h; return 2; }
nvmlReturn_t nvmlDeviceGetRemappedRows(void *d, uint *corr, uint *unc, uint *pending, uint *failed) { *corr = IDX(d); *unc = 0; *pending = 0; *failed = 0; return OK; }
nvmlReturn_t nvmlDeviceGetRetiredPages(void *d, uint cause, uint *n, uint64_t *addrs) { (void)d; (void)cause; (void)addrs; if (*n == 0) { *n = 1; return INSUFFICIENT; } *n = 1; return OK; }
nvmlReturn_t nvmlDeviceGetPcieReplayCounter(void *d, uint *n) { *n = IDX(d) == 1 ? 23 : 0; return OK; }
nvmlReturn_t nvmlDeviceGetViolationStatus(void *d, uint policy, nvmlViolationTime_t *v) { v->referenceTime = 1000000; v->violationTime = (policy == 1 && IDX(d) == 1) ? 250000 : 0; return OK; }
nvmlReturn_t nvmlDeviceGetTotalEnergyConsumption(void *d, uint64_t *mj) { (void)d; *mj = 5000000000ULL; return OK; }
nvmlReturn_t nvmlDeviceGetPerformanceState(void *d, uint *p) { (void)d; *p = 0; return OK; }
nvmlReturn_t nvmlDeviceGetTemperatureThreshold(void *d, uint type, uint *t) { (void)d; *t = type == 0 ? 95 : 90; return OK; }
nvmlReturn_t nvmlDeviceGetFieldValues(void *d, int n, nvmlFieldValue_t *f) {
    (void)d;
    for (int i = 0; i < n; i++) {
        f[i].nvmlReturn = OK; f[i].latencyUsec = 1000000; f[i].valueType = 3;
        switch (f[i].fieldId) { case 82: f[i].value = 55; break; case 90: f[i].value = 2048; break; case 91: f[i].value = 4096; break; default: f[i].nvmlReturn = NOT_SUPPORTED; }
    }
    return OK;
}
nvmlReturn_t nvmlDeviceGetComputeRunningProcesses_v3(void *d, uint *n, nvmlProcessInfo_t *p) {
    if (*n < 1) { *n = 1; return INSUFFICIENT; }
    *n = 1; p[0].pid = (uint)getpid(); p[0].usedGpuMemory = (IDX(d) == 0 ? 55ULL : 18ULL) << 30; p[0].gpuInstanceId = 0xFFFFFFFF; p[0].computeInstanceId = 0xFFFFFFFF;
    return OK;
}
nvmlReturn_t nvmlDeviceGetGraphicsRunningProcesses_v3(void *d, uint *n, nvmlProcessInfo_t *p) { (void)d; (void)p; *n = 0; return OK; }
nvmlReturn_t nvmlDeviceGetProcessUtilization(void *d, nvmlProcessUtilizationSample_t *s, uint *n, uint64_t since) {
    (void)since; if (*n < 1) { *n = 1; return INSUFFICIENT; }
    *n = 1; s[0].pid = (uint)getpid(); s[0].timeStamp = 1; s[0].smUtil = IDX(d) == 0 ? 95 : 40; s[0].memUtil = 50; s[0].encUtil = 0; s[0].decUtil = 0;
    return OK;
}
nvmlReturn_t nvmlEventSetCreate(void **set) { static int s; *set = &s; return OK; }
nvmlReturn_t nvmlDeviceRegisterEvents(void *d, uint64_t types, void *set) { (void)d; (void)types; (void)set; return OK; }
nvmlReturn_t nvmlEventSetWait_v2(void *set, nvmlEventData_t *e, uint timeout_ms) {
    (void)set;
    if (events_sent++ == 0) { e->device = &devs[1]; e->eventType = 0x8; e->eventData = 79; e->gpuInstanceId = 0; e->computeInstanceId = 0; return OK; }
    usleep(timeout_ms * 1000); return 10; /* timeout */
}
nvmlReturn_t nvmlEventSetFree(void *set) { (void)set; return OK; }
