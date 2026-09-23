class Netductor < Formula
  desc "Netductor control plane CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.71"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.71/netductor-darwin-arm64"
      sha256 "11f1c8f786d2ec43d8918621c832c9de0c2a7988e5a30ac9964c147aa1add1a8"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.71/netductor-darwin-amd64"
      sha256 "95494cac729dc7ffd8c23a842224674a62ba4d33aa3a32d5f7a5e8a8ec28cb88"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.71/netductor-linux-amd64"
      sha256 "068704bc270b977cd50df7a32640a182745164d79907971f7a5c0c18f761e9fe"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.71/netductor-linux-arm64"
      sha256 "b9c8a68f129130c504fb78f13f4b6e343d3e8e41db946cdb739bc9bf665b1bab"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
