#include <chrono>
#include <iostream>

int main() {
    using clock = std::chrono::steady_clock;
    using namespace std::chrono;

    std::cout << "Spinning for ~2 seconds...\n";

    const auto start  = clock::now();
    const auto target = start + seconds(2);

    // Busy loop until we reach target time (100% CPU on one core)
    while (clock::now() < target) {
        // nothing: pure spin
    }

    std::cout << "Done.\n";
    return 0;
}
