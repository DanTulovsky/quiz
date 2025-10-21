import { Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { useEffect } from 'react';
import { useAuth } from './hooks/useAuth';
import { useMobileDetection } from './hooks/useMobileDetection';
import { TranslationProvider } from './contexts/TranslationContext';
import { TranslationOverlay } from './components/TranslationOverlay';
import LoginPage from './pages/LoginPage';
import SignupPage from './pages/SignupPage';
import OAuthCallbackPage from './pages/OAuthCallbackPage';
import QuizPage from './pages/QuizPage';
import ReadingComprehensionPage from './pages/ReadingComprehensionPage';
import VocabularyPage from './pages/VocabularyPage';
import SnippetsPage from './pages/SnippetsPage';
import PhrasebookIndexPage from './pages/PhrasebookIndexPage';
import PhrasebookCategoryPage from './pages/PhrasebookCategoryPage';
import DailyPage from './pages/DailyPage';
import ProgressPage from './pages/ProgressPage';
import SettingsPage from './pages/SettingsPage';
import StoryPage from './pages/StoryPage';
import SavedConversationsPage from './pages/SavedConversationsPage';
import BookmarkedMessagesPage from './pages/BookmarkedMessagesPage';
import AdminPage from './pages/AdminPage';
import BackendAdminPage from './pages/admin/BackendAdminPage';
import UserManagementPage from './pages/admin/UserManagementPage';
import DataExplorerPage from './pages/admin/DataExplorerPage';
import StoryExplorerPage from './pages/admin/StoryExplorerPage';
import WorkerAdminPage from './pages/admin/WorkerAdminPage';
import AnalyticsPage from './pages/admin/AnalyticsPage';
import NotificationsPage from './pages/admin/NotificationsPage';
import DailyAdminPage from './pages/admin/DailyAdminPage';
import TranslationUsagePage from './pages/admin/TranslationUsagePage';
import NotFoundPage from './pages/NotFoundPage';
import Layout from './components/Layout';
import AdminLayout from './components/AdminLayout';
import MobileLayout from './components/MobileLayout';
import MobileLoginPage from './pages/mobile/MobileLoginPage';
import MobileSignupPage from './pages/mobile/MobileSignupPage';
import MobileQuizPage from './pages/mobile/MobileQuizPage';
import MobileVocabularyPage from './pages/mobile/MobileVocabularyPage';
import MobileReadingComprehensionPage from './pages/mobile/MobileReadingComprehensionPage';
import MobileDailyPage from './pages/mobile/MobileDailyPage';
import MobileStoryPage from './pages/mobile/MobileStoryPage';
import MobileSavedConversationsPage from './pages/mobile/MobileSavedConversationsPage';
import MobileBookmarkedMessagesPage from './pages/mobile/MobileBookmarkedMessagesPage';
import { Center, Loader } from '@mantine/core';

function App() {
  const { user, isLoading } = useAuth();
  const { isMobile } = useMobileDetection();
  const location = useLocation();

  // Mobile redirect logic
  useEffect(() => {
    if (isMobile && !location.pathname.startsWith('/m/')) {
      // Redirect to mobile version
      const mobilePath = '/m' + location.pathname + location.search;
      window.location.href = mobilePath;
    }
  }, [isMobile, location.pathname, location.search]);

  if (isLoading) {
    return (
      <Center h='100vh'>
        <Loader size='md' />
      </Center>
    );
  }

  return (
    <TranslationProvider>
      <Routes>
        <Route
          path='/login'
          element={user ? <Navigate to='/quiz' /> : <LoginPage />}
        />
        <Route
          path='/signup'
          element={user ? <Navigate to='/quiz' /> : <SignupPage />}
        />
        <Route
          path='/'
          element={user ? <Navigate to='/quiz' /> : <Navigate to='/login' />}
        />
        <Route
          path='/quiz/:questionId'
          element={
            user ? (
              <Layout>
                <QuizPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/quiz'
          element={
            user ? (
              <Layout>
                <QuizPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/vocabulary/:questionId'
          element={
            user ? (
              <Layout>
                <VocabularyPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/vocabulary'
          element={
            user ? (
              <Layout>
                <VocabularyPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/snippets'
          element={
            user ? (
              <Layout>
                <SnippetsPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/phrasebook'
          element={
            user ? (
              <Layout>
                <PhrasebookIndexPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/phrasebook/:category'
          element={
            user ? (
              <Layout>
                <PhrasebookCategoryPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/reading-comprehension/:questionId'
          element={
            user ? (
              <Layout>
                <ReadingComprehensionPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/reading-comprehension'
          element={
            user ? (
              <Layout>
                <ReadingComprehensionPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/daily/:date'
          element={
            user ? (
              <Layout>
                <DailyPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/daily'
          element={
            user ? (
              <Layout>
                <DailyPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/story'
          element={
            user ? (
              <Layout>
                <StoryPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/progress'
          element={
            user ? (
              <Layout>
                <ProgressPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/settings'
          element={
            user ? (
              <Layout>
                <SettingsPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/conversations'
          element={
            user ? (
              <Layout>
                <SavedConversationsPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/bookmarks'
          element={
            user ? (
              <Layout>
                <BookmarkedMessagesPage />
              </Layout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/admin'
          element={
            user ? (
              <AdminLayout>
                <AdminPage />
              </AdminLayout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/admin/backend/adminz'
          element={
            user ? (
              <AdminLayout>
                <BackendAdminPage />
              </AdminLayout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/admin/backend/userz'
          element={
            user ? (
              <AdminLayout>
                <UserManagementPage />
              </AdminLayout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/admin/backend/data-explorer'
          element={
            user ? (
              <AdminLayout>
                <DataExplorerPage />
              </AdminLayout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />

        <Route
          path='/admin/backend/story-explorer'
          element={
            user ? (
              <AdminLayout>
                <StoryExplorerPage />
              </AdminLayout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />

        <Route
          path='/admin/worker/adminz'
          element={
            user ? (
              <AdminLayout>
                <WorkerAdminPage />
              </AdminLayout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/admin/worker/analyticsz'
          element={
            user ? (
              <AdminLayout>
                <AnalyticsPage />
              </AdminLayout>
            ) : (
              <Navigate
                to={`/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />

        <Route
          path='/admin/worker/notifications'
          element={
            user ? (
              <AdminLayout>
                <NotificationsPage />
              </AdminLayout>
            ) : (
              <Navigate to='/login' />
            )
          }
        />
        <Route
          path='/admin/stats/translation'
          element={
            user ? (
              <AdminLayout>
                <TranslationUsagePage />
              </AdminLayout>
            ) : (
              <Navigate to='/login' />
            )
          }
        />

        <Route
          path='/admin/worker/daily'
          element={
            user ? (
              <AdminLayout>
                <DailyAdminPage />
              </AdminLayout>
            ) : (
              <Navigate to='/login' />
            )
          }
        />

        <Route path='/oauth-callback' element={<OAuthCallbackPage />} />

        {/* Mobile OAuth Callback */}
        <Route path='/m/oauth-callback' element={<OAuthCallbackPage />} />

        {/* Mobile Routes */}
        <Route
          path='/m/login'
          element={user ? <Navigate to='/m/quiz' /> : <MobileLoginPage />}
        />
        <Route
          path='/m/signup'
          element={user ? <Navigate to='/m/quiz' /> : <MobileSignupPage />}
        />
        <Route
          path='/m/quiz/:questionId'
          element={
            user ? (
              <MobileLayout>
                <MobileQuizPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/quiz'
          element={
            user ? (
              <MobileLayout>
                <MobileQuizPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/vocabulary/:questionId'
          element={
            user ? (
              <MobileLayout>
                <MobileVocabularyPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/vocabulary'
          element={
            user ? (
              <MobileLayout>
                <MobileVocabularyPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/snippets'
          element={
            user ? (
              <MobileLayout>
                <SnippetsPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/phrasebook'
          element={
            user ? (
              <MobileLayout>
                <PhrasebookIndexPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/phrasebook/:category'
          element={
            user ? (
              <MobileLayout>
                <PhrasebookCategoryPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/reading-comprehension/:questionId'
          element={
            user ? (
              <MobileLayout>
                <MobileReadingComprehensionPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/reading-comprehension'
          element={
            user ? (
              <MobileLayout>
                <MobileReadingComprehensionPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/daily/:date'
          element={
            user ? (
              <MobileLayout>
                <MobileDailyPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/daily'
          element={
            user ? (
              <MobileLayout>
                <MobileDailyPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/story'
          element={
            user ? (
              <MobileLayout>
                <MobileStoryPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/conversations'
          element={
            user ? (
              <MobileLayout>
                <MobileSavedConversationsPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />
        <Route
          path='/m/bookmarks'
          element={
            user ? (
              <MobileLayout>
                <MobileBookmarkedMessagesPage />
              </MobileLayout>
            ) : (
              <Navigate
                to={`/m/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`}
              />
            )
          }
        />

        {/* Catch-all route for 404 */}
        <Route path='*' element={<NotFoundPage />} />
      </Routes>
      <TranslationOverlay />
    </TranslationProvider>
  );
}

export default App;
