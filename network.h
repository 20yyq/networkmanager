#ifndef __go_network_h
#define __go_network_h
#include <NetworkManager.h>

typedef struct {
    guint32     flags;
    guint32     freq;
    guint32     bitrate;
    guint8      strength;
    char        *ssid;
    const char  *bssid;
    const char  *mode;
    const char  *dbus_path;
} WifiData;

struct tag_ConnData {
    const char  *id;
    const char  *uuid;
    const char  *_type;
    const char  *dbus_path;
    const char  *firmware;
    gboolean    autoconnect;
    int         priority;
    const char  *ipv4_method;
    const char  *ipv4_dns;
    const char  *ipv4_addresses;
    const char  *ipv4_gateway;
    struct tag_DevData     *dev;
};

struct tag_DevData {
    const char  *iface;
    const char  *_type;
    const char  *udi;
    const char  *driver;
    const char  *firmware;
    const char  *hw_address;
    const char  *state;
    gboolean    autoconnect;
    gboolean    real;
    gboolean    software;
    struct tag_ConnData     *conn;
};

typedef struct tag_ConnData ConnData;
typedef struct tag_DevData DevData;

typedef struct {
    const char  *ednetwork;
    const char  *ednwifi;
    const char  *edwwan;
    const char  *edwimax;
    const char  *sleep_wake;
    const char  *network_control;
    const char  *wifi_protected;
    const char  *wifi_open;
    const char  *modify_system;
    const char  *modify_own;
    const char  *modify_hostname;
    const char  *modify_dns;
    const char  *reload;
    const char  *checkpoint;
    const char  *edstatic;
    const char  *connectivity_check;
} PermissionData;

typedef struct {
    GMainLoop       *loop;
    NMClient        *client;
    uint            wifiDataLen;
    WifiData        *wifiData;
    uint            connDataLen;
    ConnData        *connData;
    uint            devDataLen;
    DevData         *devData;
    PermissionData  permission;
} clientData;
extern clientData Client;

void init();
WifiData *getWifiData(int i);
ConnData *getConnData(int i);
DevData *getDevData(int i);

// WIIF func
int wifiScanAsync(int idx);

// Devices func
int notifyDeviceMonitor(const char *iface, char **type, char **bssid, char **connId);
void removeDeviceMonitor(const char *iface);

#endif /* __go_network_h */