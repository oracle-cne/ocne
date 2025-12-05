package ignition

import (
    "github.com/coreos/ignition/v2/config/v3_4/types"
    log "github.com/sirupsen/logrus"
    "reflect"
)

func Compare(a, b *types.Config) bool {
    if a == nil || b == nil {
        if a != b {
            log.Debugf("One config is nil and the other is not: a=%#v, b=%#v", a, b)
            return false
        }
        return true
    }
    if a.Ignition.Version != b.Ignition.Version {
        log.Debugf("Ignition.Version mismatch: %q vs %q", a.Ignition.Version, b.Ignition.Version)
        return false
    }
    if !compareSecurity(a.Ignition.Security, b.Ignition.Security) {
        return false
    }
    if !compareTimeouts(a.Ignition.Timeouts, b.Ignition.Timeouts) {
        return false
    }
    if !compareProxy(a.Ignition.Proxy, b.Ignition.Proxy) {
        return false
    }
    if !compareMergeAppend(a.Ignition.Config.Merge, b.Ignition.Config.Merge, "Ignition.Config.Merge") ||
        !compareConfigReference(&a.Ignition.Config.Replace, &b.Ignition.Config.Replace) {
        return false
    }
    if !compareStorage(a.Storage, b.Storage) {
        return false
    }
    if !compareSystemd(a.Systemd, b.Systemd) {
        return false
    }
    if !comparePasswd(a.Passwd, b.Passwd) {
        return false
    }
    return true
}

func compareSecurity(a, b types.Security) bool {
    if !reflect.DeepEqual(a, b) {
        log.Debugf("Ignition.Security mismatch: %#v vs %#v", a, b)
        return false
    }
    return true
}

func compareTimeouts(a, b types.Timeouts) bool {
    if !reflect.DeepEqual(a, b) {
        log.Debugf("Ignition.Timeouts mismatch: %#v vs %#v", a, b)
        return false
    }
    return true
}

func compareProxy(a, b types.Proxy) bool {
    if !reflect.DeepEqual(a, b) {
        log.Debugf("Ignition.Proxy mismatch: %#v vs %#v", a, b)
        return false
    }
    return true
}

func compareMergeAppend(a, b []types.Resource, fname string) bool {
    if len(a) != len(b) {
        log.Debugf("%s length mismatch: %d vs %d", fname, len(a), len(b))
        return false
    }
    matchedB := make(map[int]bool)
    for i, refA := range a {
        found := false
        for j, refB := range b {
            if matchedB[j] {
                continue
            }
            if reflect.DeepEqual(refA, refB) {
                matchedB[j] = true
                found = true
                break
            }
        }
        if !found {
            log.Debugf("%s: entry at a[%d]=%#v missing in b", fname, i, refA)
            return false
        }
    }
    return true
}

func compareConfigReference(a, b *types.Resource) bool {
    if !reflect.DeepEqual(a, b) {
        log.Debugf("Ignition.Config.Replace mismatch: %#v vs %#v", a, b)
        return false
    }
    return true
}

func compareStorage(a, b types.Storage) bool {
    if !compareDirectories(a.Directories, b.Directories) {
        return false
    }
    if !compareFiles(a.Files, b.Files) {
        return false
    }
    if !compareLinks(a.Links, b.Links) {
        return false
    }
    if !compareDisks(a.Disks, b.Disks) {
        return false
    }
    if !compareRaid(a.Raid, b.Raid) {
        return false
    }
    if !compareFilesystems(a.Filesystems, b.Filesystems) {
        return false
    }
    return true
}

func compareDirectories(a, b []types.Directory) bool {
    if len(a) != len(b) {
        log.Debugf("Storage.Directories length mismatch: %d vs %d", len(a), len(b))
        return false
    }
    matchedB := make(map[int]bool)
    for _, dirA := range a {
        found := false
        for j, dirB := range b {
            if matchedB[j] {
                continue
            }
            if dirA.Node.Path == dirB.Node.Path {
                if !reflect.DeepEqual(dirA, dirB) {
                    log.Debugf("Storage.Directory mismatch at path %q: %#v vs %#v", dirA.Node.Path, dirA, dirB)
                    return false
                }
                matchedB[j] = true
                found = true
                break
            }
        }
        if !found {
            log.Debugf("Storage.Directory missing in b: path=%q", dirA.Node.Path)
            return false
        }
    }
    return true
}

func compareFiles(a, b []types.File) bool {
    if len(a) != len(b) {
        log.Debugf("Storage.Files length mismatch: %d vs %d", len(a), len(b))
        return false
    }
    matchedB := make(map[int]bool)
    for _, fileA := range a {
        found := false
        for j, fileB := range b {
            if matchedB[j] {
                continue
            }
            if fileA.Node.Path == fileB.Node.Path {
                if !reflect.DeepEqual(fileA, fileB) {
                    log.Debugf("Storage.File mismatch at path %q: %#v vs %#v", fileA.Node.Path, fileA, fileB)
                    return false
                }
                matchedB[j] = true
                found = true
                break
            }
        }
        if !found {
            log.Debugf("Storage.File missing in b: path=%q", fileA.Node.Path)
            return false
        }
    }
    return true
}

func compareLinks(a, b []types.Link) bool {
    if len(a) != len(b) {
        log.Debugf("Storage.Links length mismatch: %d vs %d", len(a), len(b))
        return false
    }
    matchedB := make(map[int]bool)
    for _, linkA := range a {
        found := false
        for j, linkB := range b {
            if matchedB[j] {
                continue
            }
            if linkA.Node.Path == linkB.Node.Path {
                if !reflect.DeepEqual(linkA, linkB) {
                    log.Debugf("Storage.Link mismatch at path %q: %#v vs %#v", linkA.Node.Path, linkA, linkB)
                    return false
                }
                matchedB[j] = true
                found = true
                break
            }
        }
        if !found {
            log.Debugf("Storage.Link missing in b: path=%q", linkA.Node.Path)
            return false
        }
    }
    return true
}

func compareDisks(a, b []types.Disk) bool {
    if len(a) != len(b) {
        log.Debugf("Storage.Disks length mismatch: %d vs %d", len(a), len(b))
        return false
    }
    matchedB := make(map[int]bool)
    for _, diskA := range a {
        found := false
        for j, diskB := range b {
            if matchedB[j] {
                continue
            }
            if diskA.Device == diskB.Device {
                if !reflect.DeepEqual(diskA, diskB) {
                    log.Debugf("Storage.Disk mismatch at device %q: %#v vs %#v", diskA.Device, diskA, diskB)
                    return false
                }
                matchedB[j] = true
                found = true
                break
            }
        }
        if !found {
            log.Debugf("Storage.Disk missing in b: device=%q", diskA.Device)
            return false
        }
    }
    return true
}

func compareRaid(a, b []types.Raid) bool {
    if len(a) != len(b) {
        log.Debugf("Storage.Raid length mismatch: %d vs %d", len(a), len(b))
        return false
    }
    matchedB := make(map[int]bool)
    for _, raidA := range a {
        found := false
        for j, raidB := range b {
            if matchedB[j] {
                continue
            }
            if raidA.Name == raidB.Name {
                if !reflect.DeepEqual(raidA, raidB) {
                    log.Debugf("Storage.Raid mismatch at name %q: %#v vs %#v", raidA.Name, raidA, raidB)
                    return false
                }
                matchedB[j] = true
                found = true
                break
            }
        }
        if !found {
            log.Debugf("Storage.Raid missing in b: name=%q", raidA.Name)
            return false
        }
    }
    return true
}

func compareFilesystems(a, b []types.Filesystem) bool {
    if len(a) != len(b) {
        log.Debugf("Storage.Filesystems length mismatch: %d vs %d", len(a), len(b))
        return false
    }
    matchedB := make(map[int]bool)
    for _, fsA := range a {
        found := false
        for j, fsB := range b {
            if matchedB[j] {
                continue
            }
            if fsA.Device == fsB.Device {
                if !reflect.DeepEqual(fsA, fsB) {
                    log.Debugf("Storage.Filesystem mismatch at name %q: %#v vs %#v", fsA.Device, fsA, fsB)
                    return false
                }
                matchedB[j] = true
                found = true
                break
            }
        }
        if !found {
            log.Debugf("Storage.Filesystem missing in b: name=%q", fsA.Device)
            return false
        }
    }
    return true
}

func compareSystemd(a, b types.Systemd) bool {
    if !compareUnits(a.Units, b.Units) {
        return false
    }
    if !compareUnitDropins(a.Units, b.Units) {
        return false
    }
    return true
}

func compareUnits(a, b []types.Unit) bool {
    if len(a) != len(b) {
        log.Debugf("Systemd.Units length mismatch: %d vs %d", len(a), len(b))
        return false
    }
    matchedB := make(map[int]bool)
    for _, unitA := range a {
        found := false
        for j, unitB := range b {
            if matchedB[j] {
                continue
            }
            if unitA.Name == unitB.Name {
                if !reflect.DeepEqual(unitA, unitB) {
                    log.Debugf("Systemd.Unit mismatch at name %q: %#v vs %#v", unitA.Name, unitA, unitB)
                    return false
                }
                matchedB[j] = true
                found = true
                break
            }
        }
        if !found {
            log.Debugf("Systemd.Unit missing in b: name=%q", unitA.Name)
            return false
        }
    }
    return true
}

func compareUnitDropins(a, b []types.Unit) bool {
    if len(a) != len(b) {
        log.Debugf("Systemd.Units length mismatch (dropins): %d vs %d", len(a), len(b))
        return false
    }
    for _, unitA := range a {
        for _, unitB := range b {
            if unitA.Name == unitB.Name {
                if len(unitA.Dropins) != len(unitB.Dropins) {
                    log.Debugf("Systemd.Unit.Dropins length mismatch for unit %q: %d vs %d", unitA.Name, len(unitA.Dropins), len(unitB.Dropins))
                    return false
                }
                matchedD := make(map[int]bool)
                for _, dropA := range unitA.Dropins {
                    found := false
                    for j, dropB := range unitB.Dropins {
                        if matchedD[j] {
                            continue
                        }
                        if dropA.Name == dropB.Name && dropA.Contents == dropB.Contents {
                            matchedD[j] = true
                            found = true
                            break
                        }
                    }
                    if !found {
                        log.Debugf("Systemd.Unit.Dropin missing or changed in b: unit=%q, dropin=%q, dropinA=%#v", unitA.Name, dropA.Name, dropA)
                        return false
                    }
                }
            }
        }
    }
    return true
}

func comparePasswd(a, b types.Passwd) bool {
    if !compareUsers(a.Users, b.Users) {
        return false
    }
    if !compareGroups(a.Groups, b.Groups) {
        return false
    }
    return true
}

func compareUsers(a, b []types.PasswdUser) bool {
    if len(a) != len(b) {
        log.Debugf("Passwd.Users length mismatch: %d vs %d", len(a), len(b))
        return false
    }
    matchedB := make(map[int]bool)
    for _, usrA := range a {
        found := false
        for j, usrB := range b {
            if matchedB[j] {
                continue
            }
            if usrA.Name == usrB.Name {
                if !reflect.DeepEqual(usrA, usrB) {
                    log.Debugf("Passwd.User mismatch at name %q: %#v vs %#v", usrA.Name, usrA, usrB)
                    return false
                }
                matchedB[j] = true
                found = true
                break
            }
        }
        if !found {
            log.Debugf("Passwd.User missing in b: name=%q", usrA.Name)
            return false
        }
    }
    return true
}

func compareGroups(a, b []types.PasswdGroup) bool {
    if len(a) != len(b) {
        log.Debugf("Passwd.Groups length mismatch: %d vs %d", len(a), len(b))
        return false
    }
    matchedB := make(map[int]bool)
    for _, grpA := range a {
        found := false
        for j, grpB := range b {
            if matchedB[j] {
                continue
            }
            if grpA.Name == grpB.Name {
                if !reflect.DeepEqual(grpA, grpB) {
                    log.Debugf("Passwd.Group mismatch at name %q: %#v vs %#v", grpA.Name, grpA, grpB)
                    return false
                }
                matchedB[j] = true
                found = true
                break
            }
        }
        if !found {
            log.Debugf("Passwd.Group missing in b: name=%q", grpA.Name)
            return false
        }
    }
    return true
}
