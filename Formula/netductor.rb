class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.93"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.93/netductor-op-darwin-arm64"
      sha256 "1e1ee603d3876464256de937577af4dda47cb6373d54a8347af087f9046b4f8b"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.93/netductor-op-darwin-amd64"
      sha256 "fa96c93dc887965b9ea5bb4abb0abc60bb79490d23f6a074b167fce84b762f4e"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.93/netductor-op-linux-amd64"
      sha256 "d0627173f47bdf10eafe4118ca058e20864eb2b8ad0fa908c659fb9b006e427b"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.93/netductor-op-linux-arm64"
      sha256 "db00c897f9a61bc66ae81bd512ded3bfedbd5f5f22c024f105b6c15735b84089"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
