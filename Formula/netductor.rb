class Netductor < Formula
  desc "Netductor control plane CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.72"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.72/netductor-darwin-arm64"
      sha256 "7d0b138637c6c995963c0cfa66a32e78fc402b08ea122680993884c85d6e614a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.72/netductor-darwin-amd64"
      sha256 "02320ca36ff9eea5c7b2ae7ea90f064c58385ad37bce09b1ae8659441df6c94b"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.72/netductor-linux-amd64"
      sha256 "3c87be87d917b184dc6768bb84a8c2516ee812cf2937a251abc51e8dec363958"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.72/netductor-linux-arm64"
      sha256 "2401efe3fc47b4052270079ee73e22261be1fbbc56f27a0e01803fb94815973c"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
