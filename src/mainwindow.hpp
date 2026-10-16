#pragma once
#include <QMainWindow>
#include <QWidget>

class QVBoxLayout;
class QHBoxLayout;
class QListWidget;
class QTextBrowser;
class QPushButton;
class QLabel;
class QLineEdit;
class QTextEdit;
class QSplitter;
class QToolButton;
class QFrame;

class FynClient;

class MainWindow : public QMainWindow {
    Q_OBJECT
public:
    explicit MainWindow(FynClient *client = nullptr, QWidget *parent = nullptr);
    void addChatMessage(const QString &text, bool incoming = true);
    QTextEdit* inputEdit() { return inputArea; }
    QLabel* headerLabel() { return chatHeader; }

signals:
    void authSucceeded();

private:
    void setupUI();
    void applyTheme();
    void styleWidgets();
    void setupMenu();
    void showAuthDialog();
    void toggleSidebar();

    FynClient *m_client;
    bool m_loggedIn;

    QWidget *iconTray;
    QWidget *sidebar;
    QWidget *chatPane;
    QListWidget *chatList;
    QLineEdit *searchBar;
    QLabel *avatarLabel;
    QLabel *chatHeader;
    QTextBrowser *chatHistory;
    QTextEdit *inputArea;
    QPushButton *sendButton;
    QToolButton *emojiButton;
    QToolButton *attachButton;
    QPushButton *toggleSidebarBtn;
    QSplitter *splitter;
    bool sidebarExpanded;
};
