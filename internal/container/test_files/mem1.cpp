#include <bits/stdc++.h>
using namespace std;

int main() {
    vector<char*> allocations;
    const size_t chunkSize = 1024 * 1024;

    while(true) {
        char *p = new(nothrow) char[chunkSize];
        if (!p) {
            break;
        }
        allocations.push_back(p);
    }

    return 0;
}
