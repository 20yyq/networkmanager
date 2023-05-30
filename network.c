/**
 * @Author: Eacher
 * @Date:   2023-05-22 10:42:36
 * @LastEditTime:   2023-05-30 14:41:20
 * @LastEditors:    Eacher
 * --------------------------------------------------------------------------------------------------------------------<
 * @Description:  gcc 编译动态库： gcc -shared -fPIC -o libgonmcli.so network.c `pkg-config --cflags --libs libnm`
 *      gcc 编译静态库： gcc -c -fPIC -o gonmcli.o network.c `pkg-config --cflags --libs libnm`  \
 *      ar rcs libgonmcli.a gonmcli.o
 * --------------------------------------------------------------------------------------------------------------------<
 * @FilePath:   /gonmcli/network.c
 */
#include "network.h"

GMainLoop   *loop   = NULL;
NMClient    *client = NULL;

gboolean init() {
    if (NULL != client)
        return FALSE;
    GError *error = NULL;
    loop = g_main_loop_new(NULL, FALSE);
    if (!(client = nm_client_new(NULL, &error))) {
        g_error_free(error);
        return FALSE;
    }
    if (!nm_client_get_nm_running(client)) {
        g_object_unref(client);
        return FALSE;
    }
    return TRUE;
}

void runLoop() {
    g_main_loop_run(loop);
    g_main_loop_unref(loop);
    g_object_unref(client);
    loop = NULL;
    client = NULL;
}

void quitLoop() {
    g_main_loop_quit(loop);
}

/*************************************************************** WIFI Start *********************************************************************/

// GoLang 实现该回调函数
extern int scanCallBackFunc(const char *name, guint n, WifiData *wd);
void wifiScanCallback(GObject* object, GAsyncResult* res, gpointer data);
void wifiScanCallback(GObject* object, GAsyncResult* res, gpointer data) {
    if (NULL != res) {
        const GPtrArray *list = nm_device_wifi_get_access_points(NM_DEVICE_WIFI(g_async_result_get_source_object(res)));
        if (1 == scanCallBackFunc("start", list->len, NULL)) {
            WifiData wd;
            for (int i = 0; i < list->len; i++) {
                NM80211Mode mode    = nm_access_point_get_mode(list->pdata[i]);
                GBytes *ssid        = nm_access_point_get_ssid(list->pdata[i]);
                if (ssid)
                    wd.ssid    = nm_utils_ssid_to_utf8(g_bytes_get_data(ssid, NULL), g_bytes_get_size(ssid));
                wd.freq        = nm_access_point_get_frequency(list->pdata[i]);
                wd.bitrate     = nm_access_point_get_max_bitrate(list->pdata[i]);
                wd.flags       = nm_access_point_get_flags(list->pdata[i]);
                wd.strength    = nm_access_point_get_strength(list->pdata[i]);
                wd.bssid       = nm_access_point_get_bssid(list->pdata[i]);
                wd.mode        = mode == NM_802_11_MODE_ADHOC   ? "Ad-Hoc" :
                                mode == NM_802_11_MODE_INFRA ? "Infrastructure" : "Unknown";
                wd.dbus_path   = nm_object_get_path(NM_OBJECT(list->pdata[i]));
                scanCallBackFunc("runFunc", (guint)i, &wd);
                if (NULL != wd.ssid) {
                    g_free(wd.ssid);
                    wd.ssid = NULL;
                }
            }
            scanCallBackFunc("close", 0, NULL);
        }
    }
}

int wifiScanAsync() {
    // libnm 版本需升级
    if (NM_CLIENT_PERMISSION_RESULT_YES != nm_client_get_permission_result(client, NM_CLIENT_PERMISSION_WIFI_SCAN)) {
        g_warning("controls whether wifi scans can not be performed \n");
        return 0; 
    }
    NMDevice *device = NULL;
    const GPtrArray *devices = nm_client_get_devices(client);
    for (int i = 0; i < devices->len; i++) {
        device = g_ptr_array_index(devices, i);
        if (NM_IS_DEVICE_WIFI(device))
            break;
        device = NULL;
    }
    if (NULL == device)
        return 0;
    nm_device_wifi_request_scan_options_async(NM_DEVICE_WIFI(device), NULL, NULL, wifiScanCallback, NULL);
    return 1;
}

/*************************************************************** WIFI End *********************************************************************/

/************************************************************* Device Start *******************************************************************/

// GoLang 实现该回调函数
extern void deviceMonitorCallBackFunc(const char *funcName, const char *devName, guint n);
void device_state(NMDevice *device, GParamSpec *pspec, NMClient *client);
void device_state(NMDevice *device, GParamSpec *pspec, NMClient *client){
    deviceMonitorCallBackFunc("device_state", nm_device_get_iface(device), (guint)nm_device_get_state(device));
}
void device_ac(NMDevice *device, GParamSpec *pspec, NMClient *client);
void device_ac(NMDevice *device, GParamSpec *pspec, NMClient *client) {
    deviceMonitorCallBackFunc("device_ac", nm_device_get_iface(device), (guint)nm_device_get_state(device));
}

void removeDeviceMonitor(const char *iface) {
    NMDevice *dev = nm_client_get_device_by_iface(client, iface);
    if (NULL != dev) {
        g_signal_handlers_disconnect_by_func(dev, device_state, client);
        g_signal_handlers_disconnect_by_func(dev, device_ac, client); 
    }
}

int notifyDeviceMonitor(const char *iface, char *type, char *bssid, char *connId) {
    NMDevice *dev = nm_client_get_device_by_iface(client, iface);
    if (NULL == dev)
        return 0;
    switch (nm_device_get_device_type(dev)) {
    case NM_DEVICE_TYPE_ETHERNET:
        type = g_strdup("ethernet");
        break;
    case NM_DEVICE_TYPE_WIFI:
        type = g_strdup("wifi");
        break;
    case NM_DEVICE_TYPE_MODEM:
        type = g_strdup("modem");
        break;
    case NM_DEVICE_TYPE_BOND:
        type = g_strdup("bond");
        break;
    case NM_DEVICE_TYPE_VLAN:
        type = g_strdup("vlan");
        break;
    case NM_DEVICE_TYPE_BRIDGE:
        type = g_strdup("bridge");
        break;
    case NM_DEVICE_TYPE_GENERIC:
        type = g_strdup("generic");
        break;
    default:
        type = g_strdup("unknown");
        break;
    }
    bssid = g_strdup(nm_device_get_hw_address(dev));
    connId = g_strdup(nm_active_connection_get_id(nm_device_get_active_connection(dev)));
    g_signal_connect (dev, "notify::" "state", G_CALLBACK(device_state), client);
    g_signal_connect (dev, "notify::" "active-connection", G_CALLBACK(device_ac), client);
    return 1;
}


/************************************************************* Device End *******************************************************************/

