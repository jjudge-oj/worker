#include <bits/stdc++.h>
using namespace std;

int main() {
    srand(time(0));

    long long sum = 0;

    for(int i = 0; i < 1e7; i++) {
        int randomNumber = rand() % 100000;
        sum += randomNumber;
    }

    cout << sum << endl;
}
