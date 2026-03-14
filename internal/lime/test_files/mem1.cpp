#include <bits/stdc++.h>
using namespace std;

int main() {
    vector<char*> allocations;
    const size_t chunkSize = 1024 * 1024;

    while (true) {
        char *p = new(nothrow) char[chunkSize];
        if (!p) {
            cerr << "new failed\n";
            break;
        }
        // Touch every page to force commit
        for (size_t i = 0; i < chunkSize; i += 4096) p[i] = 1;

        allocations.push_back(p);
    }
    return 0;
}
