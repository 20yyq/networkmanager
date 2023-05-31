#ifndef __go_network_h
#define __go_network_h
#include <NetworkManager.h>

typedef struct {
    guint32     flags;
    guint32     freq;
    guint32     bitrate;
    guint8      strength;
    char *ssid;
    const char *bssid;
    const char *mode;
    const char *dbus_path;
} WifiData;

typedef struct {
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
} ConnData;

typedef struct {
    const char  *iface;
    const char  *_type;
    const char  *udi;
    const char  *driver;
    const char  *firmware;
    const char  *hw_address;
    const char  *state;
    const char  *uuid;
    gboolean    autoconnect;
    gboolean    real;
    gboolean    software;
} DevData;

gboolean init();
void runLoop();
void quitLoop();

// WIIF func
int wifiScanAsync();

// Devices func
int notifyDeviceMonitor(const char *iface, char **type, char **bssid, char **connId);
void removeDeviceMonitor(const char *iface);

#endif /* __go_network_h */