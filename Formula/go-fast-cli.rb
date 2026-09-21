class GoFastCli < Formula
  desc "Command-line tool to test internet speed using Fast.com with a live TUI"
  homepage "https://github.com/orekasep/go-fast-cli"
  version "1.0.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/orekasep/go-fast-cli/releases/download/v1.0.0/go-fast-cli_v1.0.0_darwin-arm64.tar.gz"
      sha256 "c54f966a326b5a32b3b1d47c27019a1a2ec7ba4708847c3815b918de5bbbbd8f"

      def install
        bin.install "fast" => "go-fast-cli"
        bin.install_symlink bin/"go-fast-cli" => "fast"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/orekasep/go-fast-cli/releases/download/v1.0.0/go-fast-cli_v1.0.0_darwin-amd64.tar.gz"
      sha256 "1b6e1d552b78b820bf51f8af2bfe24986b0df079f4e6fa2e1020417b1b761932"

      def install
        bin.install "fast" => "go-fast-cli"
        bin.install_symlink bin/"go-fast-cli" => "fast"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/orekasep/go-fast-cli/releases/download/v1.0.0/go-fast-cli_v1.0.0_linux-arm64.tar.gz"
      sha256 "8379bb1414cd99d2060ebe537bee6a717273c4ae0ecc6f16566550d93679bbb6"

      def install
        bin.install "fast" => "go-fast-cli"
        bin.install_symlink bin/"go-fast-cli" => "fast"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/orekasep/go-fast-cli/releases/download/v1.0.0/go-fast-cli_v1.0.0_linux-amd64.tar.gz"
      sha256 "adba34b36459d7ae89e27bdc84c7feac195d59019463f9ca35c6bd5c789ced05"

      def install
        bin.install "fast" => "go-fast-cli"
        bin.install_symlink bin/"go-fast-cli" => "fast"
      end
    end
  end

  test do
    assert_match "go-fast-cli", shell_output("#{bin}/go-fast-cli -version")
  end
end
