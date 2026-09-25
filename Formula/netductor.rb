class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.13"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.13/netductor-op-darwin-arm64"
      sha256 "5129165e512cb426250586b9b7828578fb07d221b45e05c55ec01cb0edf7fa28"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.13/netductor-op-darwin-amd64"
      sha256 "f9b81caf37eaf3d9e80b2a7ac5c1f57f4c61650a3826c13714bc3219cc86bbb3"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.13/netductor-op-linux-amd64"
      sha256 "6248aed5049b4f79b2286010ccb3314dfb91d69cc8905622963642c92f9e7651"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.13/netductor-op-linux-arm64"
      sha256 "7abadfa8fda1abb96b2efa0881309554af13081aabc370de218c07fd3bbdc694"
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
