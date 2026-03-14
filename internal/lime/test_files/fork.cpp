#include <bits/stdc++.h>
using namespace std;

int main() {
    while(true) {
        pid_t pid = fork();
        if(pid == -1) {
            cout << "cannot fork" << endl;
            break;
        }
    }

    return 0;
}
