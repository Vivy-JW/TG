using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Text;
using System.Threading;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Markup;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon;

namespace SpecDesignerPreference
{
	// Token: 0x020000EB RID: 235
	public partial class PreferenceWindow : Window
	{
		// Token: 0x060007E0 RID: 2016 RVA: 0x00023148 File Offset: 0x00021348
		public PreferenceWindow()
		{
			this.InitializeComponent();
			if (!string.IsNullOrEmpty(SettingManager.Get().Info_Languages))
			{
				this.LangGrp.Visibility = Visibility.Visible;
				this.langlist.ItemsSource = LanguageModel.Load();
				this.langlist.DisplayMemberPath = "Name";
				this.langlist.SelectedValuePath = "Lang";
			}
			else
			{
				this.LangGrp.Visibility = Visibility.Hidden;
			}
			this.element = new PreferenceModel(PreferenceManager.Current.Settings.ToString());
			base.DataContext = this.element;
			this.CommonSection = PreferenceManager.Current.Settings.CommonUsedSetting;
			this.CommonUsedWindow.DataContext = this.CommonSection;
			this.cboMaxRecentList.ItemsSource = MaxRecentFilesModel.Load();
			this.cboMaxRecentList.DisplayMemberPath = "MaxKey";
			this.cboMaxRecentList.SelectedValuePath = "MaxValue";
		}

		// Token: 0x060007E1 RID: 2017 RVA: 0x00023238 File Offset: 0x00021438
		private void ReloadButton_Click(object sender, RoutedEventArgs e)
		{
			PreferenceManager.Current.Reset();
			this.element = PreferenceManager.Current.Settings;
			base.DataContext = this.element;
			this.CommonSection = PreferenceManager.Current.Settings.CommonUsedSetting;
			this.CommonUsedWindow.DataContext = this.CommonSection;
		}

		// Token: 0x060007E2 RID: 2018 RVA: 0x00023294 File Offset: 0x00021494
		private void SaveButton_Click(object sender, RoutedEventArgs e)
		{
			PreferenceManager.Current.Save(this.element);
			if (!PreferenceManager.Current.Settings.UILang.Equals(Thread.CurrentThread.CurrentUICulture.Name))
			{
				StringBuilder stringBuilder = new StringBuilder();
				stringBuilder.AppendLine(Application.Current.FindResource("Message_ChangeLanguageWarning") as string).AppendLine();
				stringBuilder.AppendLine(Application.Current.FindResource("Message_ChangeLanguageWarning1") as string);
				stringBuilder.AppendLine(Application.Current.FindResource("Message_ChangeLanguageWarning2") as string);
				stringBuilder.AppendLine(Application.Current.FindResource("Message_ChangeLanguageWarning3") as string);
				stringBuilder.AppendLine(Application.Current.FindResource("Message_ChangeLanguageWarning4") as string);
				DesignerMessageBox.Show(stringBuilder.ToString());
				this.UnsupportLang(PreferenceManager.Current.Settings.UILang);
			}
			base.Close();
		}

		// Token: 0x060007E3 RID: 2019 RVA: 0x00023393 File Offset: 0x00021593
		private void CancelButton_Click(object sender, RoutedEventArgs e)
		{
			base.Close();
		}

		// Token: 0x060007E4 RID: 2020 RVA: 0x0002339B File Offset: 0x0002159B
		private void ChangeBackgroundColorButton_Click(object sender, RoutedEventArgs e)
		{
		}

		// Token: 0x060007E5 RID: 2021 RVA: 0x0002339D File Offset: 0x0002159D
		private void langlist_SelectionChanged(object sender, DataTransferEventArgs e)
		{
		}

		// Token: 0x060007E6 RID: 2022 RVA: 0x000233A0 File Offset: 0x000215A0
		private bool UnsupportLang(string langName)
		{
			bool flag = false;
			string text = string.Empty;
			try
			{
				string text2 = "/SpecDesignerCommon;component/Langs/" + langName + ".xaml";
				Application.LoadComponent(new Uri(text2, UriKind.RelativeOrAbsolute));
				flag = true;
			}
			catch (Exception)
			{
				text = Application.Current.FindResource("Message_UnsupportedLanguage") as string;
				DesignerMessageBox.Show(text);
				flag = false;
			}
			return flag;
		}

		// Token: 0x060007E7 RID: 2023 RVA: 0x0002340C File Offset: 0x0002160C
		private void SimpleDownload_Click(object sender, RoutedEventArgs e)
		{
			if (this.SimpleDownload.IsChecked != PreferenceManager.Current.Settings.SimpleDownloadDialog)
			{
				DesignerMessageBox.Show(Application.Current.FindResource("Preference_SimpleDownloadDialog_Change") as string);
			}
		}

		// Token: 0x040002C7 RID: 711
		private PreferenceModel element;

		// Token: 0x040002C8 RID: 712
		private CommonUsedElement CommonSection;
	}
}
