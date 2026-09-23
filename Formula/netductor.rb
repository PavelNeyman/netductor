class Netductor < Formula
  desc "Netductor control plane CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.73"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.73/netductor-darwin-arm64"
      sha256 "c368ffc1d9bd89cbc061aa8a4bc7d91101583765da66941bf218dfaaeba692f0"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.73/netductor-darwin-amd64"
      sha256 "ea7711d75dc0951bca2d3b50d68467a8648da73600602f3d2c73ad91bf5b1605"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.73/netductor-linux-amd64"
      sha256 "b05fd72b7488178fcd1f8b709157a51383f036c63d92e153d46aab94e044ab5b"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.73/netductor-linux-arm64"
      sha256 "8ba4510ebcf0cdc9552f1776bef3af26bfe1df70b6383b626ab31fdf0314bd9e"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
