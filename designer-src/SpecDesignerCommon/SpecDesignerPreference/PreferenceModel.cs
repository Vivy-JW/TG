using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Globalization;
using System.Linq;
using System.Text;
using System.Text.RegularExpressions;
using System.Windows.Media;
using System.Xml.Linq;

namespace SpecDesignerPreference
{
	// Token: 0x02000085 RID: 133
	public class PreferenceModel : INotifyPropertyChanged
	{
		// Token: 0x14000015 RID: 21
		// (add) Token: 0x0600053E RID: 1342 RVA: 0x00018250 File Offset: 0x00016450
		// (remove) Token: 0x0600053F RID: 1343 RVA: 0x00018288 File Offset: 0x00016488
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000540 RID: 1344 RVA: 0x000182BD File Offset: 0x000164BD
		private void OnPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x06000541 RID: 1345 RVA: 0x000182DC File Offset: 0x000164DC
		public PreferenceModel()
		{
			this.CommonUsedSetting = new CommonUsedElement();
		}

		// Token: 0x06000542 RID: 1346 RVA: 0x000183C0 File Offset: 0x000165C0
		public PreferenceModel(string content)
		{
			this.Parse(XElement.Parse(content));
		}

		// Token: 0x06000543 RID: 1347 RVA: 0x000184A4 File Offset: 0x000166A4
		private void Parse(XElement source)
		{
			if (source.Element("FontSize") != null)
			{
				int num = 0;
				int.TryParse(source.Element("FontSize").Value, out num);
				if (num > 0)
				{
					this.FontSize = num;
				}
			}
			if (source.Element("FontFamily") != null)
			{
				this.FontFamily = new FontFamily(source.Element("FontFamily").Value);
			}
			if (source.Element("TabWidth") != null)
			{
				int num2 = 0;
				if (int.TryParse(source.Element("TabWidth").Value, out num2))
				{
					this.TabWidth = num2;
				}
			}
			if (source.Element("WordWrap") != null)
			{
				bool flag = false;
				if (bool.TryParse(source.Element("WordWrap").Value, out flag))
				{
					this.WordWrap = flag;
				}
			}
			if (source.Element("CodeCompletion") != null)
			{
				bool flag2 = false;
				if (bool.TryParse(source.Element("CodeCompletion").Value, out flag2))
				{
					this.CodeCompletion = flag2;
				}
			}
			if (source.Element("UILang") != null && !string.IsNullOrEmpty(source.Element("UILang").Value))
			{
				this._UILang = source.Element("UILang").Value;
			}
			else
			{
				this._UILang = CultureInfo.CurrentUICulture.Name;
			}
			if (source.Element("MaxRecentFiles") != null)
			{
				int num3 = 10;
				if (int.TryParse(source.Element("MaxRecentFiles").Value, out num3))
				{
					this.MaxRecentFiles = num3;
				}
			}
			this._selfKeywordSource = ((source.Element("SelfKeyword") == null) ? string.Empty : source.Element("SelfKeyword").Value);
			this._favoriteKeywordSource = ((source.Element("FavoriteKeyword") == null) ? string.Empty : source.Element("FavoriteKeyword").Value);
			if (source.Element("Theme") != null)
			{
				ThemeOptions themeOptions = ThemeOptions.Default;
				if (Enum.TryParse<ThemeOptions>(source.Element("Theme").Value, out themeOptions))
				{
					this.Theme = themeOptions;
				}
			}
			this.Background = ((source.Element("Background") == null) ? string.Empty : source.Element("Background").Value);
			this.Foreground = ((source.Element("Foreground") == null) ? string.Empty : source.Element("Foreground").Value);
			if (source.Element("UndoTimes") != null)
			{
				int num4 = 0;
				if (int.TryParse(source.Element("UndoTimes").Value, out num4))
				{
					this.UndoTimes = num4;
				}
			}
			if (source.Element("BackupLimit") != null)
			{
				int num5 = 0;
				if (int.TryParse(source.Element("BackupLimit").Value, out num5))
				{
					this.BackupLimit = num5;
				}
			}
			if (source.Element("ValidateForm") != null)
			{
				bool flag3 = false;
				if (bool.TryParse(source.Element("ValidateForm").Value, out flag3))
				{
					this.ValidateForm = flag3;
				}
			}
			if (source.Element("TabIndexSort") != null)
			{
				bool flag4 = false;
				if (bool.TryParse(source.Element("TabIndexSort").Value, out flag4))
				{
					this.TabIndexSort = flag4;
				}
			}
			if (source.Element("ShowHint") != null)
			{
				bool flag5 = false;
				if (bool.TryParse(source.Element("ShowHint").Value, out flag5))
				{
					this.ShowHint = flag5;
				}
			}
			if (source.Element("RemaindSave") != null)
			{
				bool flag6 = false;
				if (bool.TryParse(source.Element("RemaindSave").Value, out flag6))
				{
					this.RemaindSave = flag6;
				}
			}
			if (source.Element("SimpleDownloadDialog") != null)
			{
				bool flag7 = false;
				if (bool.TryParse(source.Element("SimpleDownloadDialog").Value, out flag7))
				{
					this.SimpleDownloadDialog = flag7;
					this.StandardView = !flag7;
				}
			}
			if (source.Element("RemaindDiffSave") != null)
			{
				bool flag8 = false;
				if (bool.TryParse(source.Element("RemaindDiffSave").Value, out flag8))
				{
					this.RemaindDiffSave = flag8;
				}
			}
			if (source.Element("AlwaysSaveDiff") != null)
			{
				bool flag9 = false;
				if (bool.TryParse(source.Element("AlwaysSaveDiff").Value, out flag9))
				{
					this.AlwaysSaveDiff = flag9;
				}
			}
			if (source.Element("SimplifyColor") != null)
			{
				bool flag10 = false;
				if (bool.TryParse(source.Element("SimplifyColor").Value, out flag10))
				{
					this.SimplifyColor = flag10;
				}
			}
			if (source.Element("HorizontalArrangement") != null)
			{
				bool flag11 = false;
				if (bool.TryParse(source.Element("HorizontalArrangement").Value, out flag11))
				{
					this.HorizontalArrangement = flag11;
				}
			}
			if (source.Element("DiffComment") != null)
			{
				bool flag12 = false;
				if (bool.TryParse(source.Element("DiffComment").Value, out flag12))
				{
					this.DiffComment = flag12;
				}
			}
			if (source.Element("ViMode") != null)
			{
				bool flag13 = false;
				if (bool.TryParse(source.Element("ViMode").Value, out flag13))
				{
					this.ViMode = flag13;
				}
			}
			if (source.Element("TopstdEditPermission") != null)
			{
				bool flag14 = false;
				if (bool.TryParse(source.Element("TopstdEditPermission").Value, out flag14))
				{
					this.TopstdEditPermission = flag14;
				}
			}
			this.ServiceCloudLogin = ((source.Element("ServiceCloudLogin") == null) ? string.Empty : source.Element("ServiceCloudLogin").Value);
			this.ServiceCloudPassword = ((source.Element("ServiceCloudPassword") == null) ? string.Empty : Encoding.UTF8.GetString(Convert.FromBase64String(source.Element("ServiceCloudPassword").Value)));
			if (source.Element("DiffWithoutSection") != null)
			{
				bool flag15 = false;
				if (bool.TryParse(source.Element("DiffWithoutSection").Value, out flag15))
				{
					this.DiffWithoutSection = flag15;
				}
			}
			if (source.Element("DiffWithANSI") != null)
			{
				bool flag16 = false;
				if (bool.TryParse(source.Element("DiffWithANSI").Value, out flag16))
				{
					this.DiffWithANSI = flag16;
				}
			}
			if (source.Element("StartShowWelcome") != null)
			{
				bool flag17 = false;
				if (bool.TryParse(source.Element("StartShowWelcome").Value, out flag17))
				{
					this.StartShowWelcome = flag17;
				}
			}
			if (source.Element("DisplayOpenedFiles") != null)
			{
				bool flag18 = false;
				if (bool.TryParse(source.Element("DisplayOpenedFiles").Value, out flag18))
				{
					this.DisplayOpenedFiles = flag18;
				}
			}
			if (source.Element("ShowExample") != null)
			{
				bool flag19 = false;
				if (bool.TryParse(source.Element("ShowExample").Value, out flag19))
				{
					this.ShowExample = flag19;
				}
			}
			if (source.Element("MaxAmount") != null)
			{
				int num6 = 0;
				if (int.TryParse(source.Element("MaxAmount").Value, out num6))
				{
					this.MaxAmount = num6;
				}
			}
			if (source.Element("LocalizedProperty") != null)
			{
				bool flag20 = false;
				if (bool.TryParse(source.Element("LocalizedProperty").Value, out flag20))
				{
					this.LocalizedProperty = flag20;
				}
			}
			this.CommonUsedSetting = CommonUsedElement.Parse(source);
		}

		// Token: 0x06000544 RID: 1348 RVA: 0x00018CD4 File Offset: 0x00016ED4
		public XElement ToXML()
		{
			XElement xelement = new XElement("Preference", new object[]
			{
				new XElement("FontSize", this.FontSize.ToString()),
				new XElement("FontFamily", this.FontFamily.ToString()),
				new XElement("TabWidth", this.TabWidth.ToString()),
				new XElement("WordWrap", this.WordWrap.ToString()),
				new XElement("CodeCompletion", this.CodeCompletion.ToString()),
				new XElement("SelfKeyword", this.SelfKeyword),
				new XElement("FavoriteKeyword", this.FavoriteKeyword),
				new XElement("Theme", this.Theme.ToString()),
				new XElement("Background", this.Background),
				new XElement("Foreground", this.Foreground),
				new XElement("UndoTimes", this.UndoTimes.ToString()),
				new XElement("BackupLimit", this.BackupLimit.ToString()),
				new XElement("ValidateForm", this.ValidateForm.ToString()),
				new XElement("ShowHint", this.ShowHint.ToString()),
				new XElement("RemaindSave", this.RemaindSave.ToString()),
				new XElement("SimpleDownloadDialog", this.SimpleDownloadDialog.ToString()),
				new XElement("RemaindDiffSave", this.RemaindDiffSave.ToString()),
				new XElement("AlwaysSaveDiff", this.AlwaysSaveDiff.ToString()),
				new XElement("SimplifyColor", this.SimplifyColor.ToString()),
				new XElement("HorizontalArrangement", this.HorizontalArrangement.ToString()),
				new XElement("DiffComment", this.DiffComment.ToString()),
				new XElement("ViMode", this.ViMode.ToString()),
				new XElement("TopstdEditPermission", this.TopstdEditPermission.ToString()),
				new XElement("ServiceCloudLogin", this.ServiceCloudLogin.ToString()),
				new XElement("ServiceCloudPassword", Convert.ToBase64String(Encoding.UTF8.GetBytes(this.ServiceCloudPassword.ToString()))),
				new XElement("UILang", this.UILang),
				new XElement("MaxRecentFiles", this.MaxRecentFiles.ToString()),
				new XElement("DiffWithoutSection", this.DiffWithoutSection),
				new XElement("DiffWithANSI", this.DiffWithANSI),
				new XElement("TabIndexSort", this.TabIndexSort),
				new XElement("StartShowWelcome", this.StartShowWelcome.ToString()),
				new XElement("DisplayOpenedFiles", this.DisplayOpenedFiles.ToString()),
				new XElement("ShowExample", this.ShowExample.ToString()),
				new XElement("MaxAmount", this.MaxAmount.ToString()),
				new XElement("LocalizedProperty", this.LocalizedProperty.ToString())
			});
			xelement.Add(this.CommonUsedSetting.ToXML());
			return xelement;
		}

		// Token: 0x06000545 RID: 1349 RVA: 0x00019172 File Offset: 0x00017372
		public override string ToString()
		{
			return this.ToXML().ToString();
		}

		// Token: 0x06000546 RID: 1350 RVA: 0x00019180 File Offset: 0x00017380
		internal void Reset()
		{
			PreferenceModel preferenceModel = new PreferenceModel();
			this.FontSize = preferenceModel.FontSize;
			this.FontFamily = preferenceModel.FontFamily;
			this.TabWidth = preferenceModel.TabWidth;
			this.WordWrap = preferenceModel.WordWrap;
			this.CodeCompletion = preferenceModel.CodeCompletion;
			this.SelfKeyword = preferenceModel.SelfKeyword;
			this._selfKeywords.Clear();
			this.FavoriteKeyword = preferenceModel.FavoriteKeyword;
			this._favoriteKeyword.Clear();
			this.Theme = preferenceModel.Theme;
			this.Background = preferenceModel.Background;
			this.Foreground = preferenceModel.Foreground;
			this.UndoTimes = preferenceModel.UndoTimes;
			this.BackupLimit = preferenceModel.BackupLimit;
			this.ValidateForm = preferenceModel.ValidateForm;
			this.ShowHint = preferenceModel.ShowHint;
			this.RemaindSave = preferenceModel.RemaindSave;
			this.SimpleDownloadDialog = preferenceModel.SimpleDownloadDialog;
			this.RemaindDiffSave = preferenceModel.RemaindDiffSave;
			this.AlwaysSaveDiff = preferenceModel.AlwaysSaveDiff;
			this.SimplifyColor = preferenceModel.SimplifyColor;
			this.HorizontalArrangement = preferenceModel.HorizontalArrangement;
			this.DiffComment = preferenceModel.DiffComment;
			this.ViMode = preferenceModel.ViMode;
			this.TopstdEditPermission = preferenceModel.TopstdEditPermission;
			this.ServiceCloudLogin = preferenceModel.ServiceCloudLogin;
			this.ServiceCloudPassword = preferenceModel.ServiceCloudPassword;
			this.UILang = preferenceModel.UILang;
			this.MaxRecentFiles = preferenceModel.MaxRecentFiles;
			this.DiffWithoutSection = preferenceModel.DiffWithoutSection;
			this.DiffWithANSI = preferenceModel.DiffWithANSI;
			this.TabIndexSort = preferenceModel.TabIndexSort;
			this.StartShowWelcome = preferenceModel.StartShowWelcome;
			this.DisplayOpenedFiles = preferenceModel.DisplayOpenedFiles;
			this.ShowExample = preferenceModel.ShowExample;
			this.MaxAmount = preferenceModel.MaxAmount;
			this.LocalizedProperty = preferenceModel.LocalizedProperty;
			this.CommonUsedSetting.Clear();
		}

		// Token: 0x17000178 RID: 376
		// (get) Token: 0x06000547 RID: 1351 RVA: 0x00019358 File Offset: 0x00017558
		// (set) Token: 0x06000548 RID: 1352 RVA: 0x00019360 File Offset: 0x00017560
		public int FontSize
		{
			get
			{
				return this._fontSize;
			}
			set
			{
				if (this._fontSize == value)
				{
					return;
				}
				this._fontSize = value;
				this.OnPropertyChanged("FontSize");
			}
		}

		// Token: 0x17000179 RID: 377
		// (get) Token: 0x06000549 RID: 1353 RVA: 0x0001937E File Offset: 0x0001757E
		// (set) Token: 0x0600054A RID: 1354 RVA: 0x00019386 File Offset: 0x00017586
		public FontFamily FontFamily
		{
			get
			{
				return this._fontFamily;
			}
			set
			{
				if (this._fontFamily == value)
				{
					return;
				}
				this._fontFamily = value;
				this.OnPropertyChanged("FontFamily");
			}
		}

		// Token: 0x1700017A RID: 378
		// (get) Token: 0x0600054B RID: 1355 RVA: 0x000193A4 File Offset: 0x000175A4
		// (set) Token: 0x0600054C RID: 1356 RVA: 0x000193AC File Offset: 0x000175AC
		public int TabWidth
		{
			get
			{
				return this._tabWidth;
			}
			set
			{
				if (this._tabWidth == value)
				{
					return;
				}
				this._tabWidth = value;
				this.OnPropertyChanged("TabWidth");
			}
		}

		// Token: 0x1700017B RID: 379
		// (get) Token: 0x0600054D RID: 1357 RVA: 0x000193CA File Offset: 0x000175CA
		// (set) Token: 0x0600054E RID: 1358 RVA: 0x000193D2 File Offset: 0x000175D2
		public bool WordWrap
		{
			get
			{
				return this._wordWrap;
			}
			set
			{
				if (this._wordWrap == value)
				{
					return;
				}
				this._wordWrap = value;
				this.OnPropertyChanged("WordWrap");
			}
		}

		// Token: 0x1700017C RID: 380
		// (get) Token: 0x0600054F RID: 1359 RVA: 0x000193F0 File Offset: 0x000175F0
		// (set) Token: 0x06000550 RID: 1360 RVA: 0x000193F8 File Offset: 0x000175F8
		public bool CodeCompletion
		{
			get
			{
				return this._codeCompletion;
			}
			set
			{
				if (this._codeCompletion == value)
				{
					return;
				}
				this._codeCompletion = value;
				this.OnPropertyChanged("CodeCompletion");
			}
		}

		// Token: 0x1700017D RID: 381
		// (get) Token: 0x06000551 RID: 1361 RVA: 0x00019418 File Offset: 0x00017618
		// (set) Token: 0x06000552 RID: 1362 RVA: 0x00019467 File Offset: 0x00017667
		public string SelfKeyword
		{
			get
			{
				if (this._selfKeywords == null)
				{
					this._selfKeywords = this._selfKeywordSource.Split(new char[] { ',' }).ToList<string>();
					this._selfKeywords.Remove("");
				}
				return this._selfKeywordSource;
			}
			private set
			{
				this._selfKeywordSource = value;
				this.OnPropertyChanged("SelfKeyword");
			}
		}

		// Token: 0x06000553 RID: 1363 RVA: 0x0001947C File Offset: 0x0001767C
		public void AddSelfKeyword(string name)
		{
			if (this._selfKeywords == null)
			{
				this._selfKeywords = new List<string>();
			}
			if (this._selfKeywords.Contains(name))
			{
				return;
			}
			this._selfKeywords.Add(name);
			this.SelfKeyword = string.Join(",", this._selfKeywords);
			PreferenceManager.Current.Save();
		}

		// Token: 0x06000554 RID: 1364 RVA: 0x000194D8 File Offset: 0x000176D8
		public void RemoveSelfKeyword(string name)
		{
			if (this._selfKeywords == null)
			{
				return;
			}
			if (!this._selfKeywords.Contains(name))
			{
				return;
			}
			this._selfKeywords.Remove(name);
			this.SelfKeyword = string.Join(",", this._selfKeywords);
			PreferenceManager.Current.Save();
		}

		// Token: 0x06000555 RID: 1365 RVA: 0x0001952A File Offset: 0x0001772A
		public IEnumerable<string> GetSelfKeyword()
		{
			if (this.SelfKeyword.Length == 0 && this.FavoriteKeyword.Length == 0)
			{
				return null;
			}
			return this._selfKeywords;
		}

		// Token: 0x1700017E RID: 382
		// (get) Token: 0x06000556 RID: 1366 RVA: 0x00019550 File Offset: 0x00017750
		// (set) Token: 0x06000557 RID: 1367 RVA: 0x0001959F File Offset: 0x0001779F
		public string FavoriteKeyword
		{
			get
			{
				if (this._favoriteKeyword == null)
				{
					this._favoriteKeyword = this._favoriteKeywordSource.Split(new char[] { ',' }).ToList<string>();
					this._favoriteKeyword.Remove("");
				}
				return this._favoriteKeywordSource;
			}
			private set
			{
				this._favoriteKeywordSource = value;
				this.OnPropertyChanged("FavoriteKeyword");
			}
		}

		// Token: 0x06000558 RID: 1368 RVA: 0x000195B4 File Offset: 0x000177B4
		public void AddFavoriteKeyword(string name)
		{
			if (this._favoriteKeyword == null)
			{
				this._favoriteKeyword = new List<string>();
			}
			if (this._favoriteKeyword.Contains(name))
			{
				return;
			}
			this._favoriteKeyword.Add(name);
			this.FavoriteKeyword = string.Join(",", this._favoriteKeyword);
			PreferenceManager.Current.Save();
		}

		// Token: 0x06000559 RID: 1369 RVA: 0x00019610 File Offset: 0x00017810
		public void RemoveFavoriteKeyword(string name)
		{
			if (this._favoriteKeyword == null)
			{
				return;
			}
			if (!this._favoriteKeyword.Contains(name))
			{
				return;
			}
			this._favoriteKeyword.Remove(name);
			this.FavoriteKeyword = string.Join(",", this._favoriteKeyword);
			PreferenceManager.Current.Save();
		}

		// Token: 0x0600055A RID: 1370 RVA: 0x00019662 File Offset: 0x00017862
		public bool IsFavoriteKeywordExist(string name)
		{
			return !string.IsNullOrWhiteSpace(this.FavoriteKeyword) && this._favoriteKeyword != null && this._favoriteKeyword.Contains(name);
		}

		// Token: 0x1700017F RID: 383
		// (get) Token: 0x0600055B RID: 1371 RVA: 0x00019687 File Offset: 0x00017887
		// (set) Token: 0x0600055C RID: 1372 RVA: 0x00019690 File Offset: 0x00017890
		public ThemeOptions Theme
		{
			get
			{
				return this._theme;
			}
			set
			{
				this._theme = value;
				this.OnPropertyChanged("Theme");
				if (value == ThemeOptions.Dark)
				{
					this.Foreground = "FFFFFF";
					return;
				}
				this.Foreground = "000000";
			}
		}

		// Token: 0x17000180 RID: 384
		// (get) Token: 0x0600055D RID: 1373 RVA: 0x000196CC File Offset: 0x000178CC
		// (set) Token: 0x0600055E RID: 1374 RVA: 0x000196FB File Offset: 0x000178FB
		public string Background
		{
			get
			{
				string text = this._background;
				if (text == "FF0000")
				{
					text = (this.Background = "FFFFFF");
				}
				return text;
			}
			set
			{
				if (!Regex.IsMatch(value, this.COLOR_CHECK))
				{
					return;
				}
				this._background = value;
				this.OnPropertyChanged("Background");
			}
		}

		// Token: 0x17000181 RID: 385
		// (get) Token: 0x0600055F RID: 1375 RVA: 0x00019720 File Offset: 0x00017920
		// (set) Token: 0x06000560 RID: 1376 RVA: 0x0001974F File Offset: 0x0001794F
		public string Foreground
		{
			get
			{
				string text = this._foreground;
				if (text == "FF0000")
				{
					text = (this.Foreground = "000000");
				}
				return text;
			}
			set
			{
				if (!Regex.IsMatch(value, this.COLOR_CHECK))
				{
					return;
				}
				this._foreground = value;
				this.OnPropertyChanged("Foreground");
			}
		}

		// Token: 0x17000182 RID: 386
		// (get) Token: 0x06000561 RID: 1377 RVA: 0x00019772 File Offset: 0x00017972
		// (set) Token: 0x06000562 RID: 1378 RVA: 0x0001977A File Offset: 0x0001797A
		public bool ShowHint
		{
			get
			{
				return this._showHint;
			}
			set
			{
				if (this._showHint == value)
				{
					return;
				}
				this._showHint = value;
				this.OnPropertyChanged("ShowHint");
			}
		}

		// Token: 0x17000183 RID: 387
		// (get) Token: 0x06000563 RID: 1379 RVA: 0x00019798 File Offset: 0x00017998
		// (set) Token: 0x06000564 RID: 1380 RVA: 0x000197A0 File Offset: 0x000179A0
		public bool SimplifyColor
		{
			get
			{
				return this._simplifyColor;
			}
			set
			{
				if (this._simplifyColor == value)
				{
					return;
				}
				this._simplifyColor = value;
				this.OnPropertyChanged("SimplifyColor");
			}
		}

		// Token: 0x17000184 RID: 388
		// (get) Token: 0x06000565 RID: 1381 RVA: 0x000197BE File Offset: 0x000179BE
		// (set) Token: 0x06000566 RID: 1382 RVA: 0x000197C6 File Offset: 0x000179C6
		public bool HorizontalArrangement
		{
			get
			{
				return this._horizontalArrangement;
			}
			set
			{
				if (this._horizontalArrangement == value)
				{
					return;
				}
				this._horizontalArrangement = value;
				this.OnPropertyChanged("HorizontalArrangement");
			}
		}

		// Token: 0x17000185 RID: 389
		// (get) Token: 0x06000567 RID: 1383 RVA: 0x000197E4 File Offset: 0x000179E4
		// (set) Token: 0x06000568 RID: 1384 RVA: 0x000197EC File Offset: 0x000179EC
		public bool DiffComment
		{
			get
			{
				return this._diffComment;
			}
			set
			{
				if (this._diffComment == value)
				{
					return;
				}
				this._diffComment = value;
				this.OnPropertyChanged("DiffComment");
			}
		}

		// Token: 0x17000186 RID: 390
		// (get) Token: 0x06000569 RID: 1385 RVA: 0x0001980A File Offset: 0x00017A0A
		// (set) Token: 0x0600056A RID: 1386 RVA: 0x00019812 File Offset: 0x00017A12
		public bool TopstdEditPermission
		{
			get
			{
				return this._topstdEditPermission;
			}
			set
			{
				if (this._topstdEditPermission == value)
				{
					return;
				}
				this._topstdEditPermission = value;
				this.OnPropertyChanged("TopstdEditPermission");
			}
		}

		// Token: 0x17000187 RID: 391
		// (get) Token: 0x0600056B RID: 1387 RVA: 0x00019830 File Offset: 0x00017A30
		// (set) Token: 0x0600056C RID: 1388 RVA: 0x00019838 File Offset: 0x00017A38
		public bool ViMode
		{
			get
			{
				return this._viMode;
			}
			set
			{
				if (this._viMode == value)
				{
					return;
				}
				this._viMode = value;
				this.OnPropertyChanged("ViMode");
			}
		}

		// Token: 0x17000188 RID: 392
		// (get) Token: 0x0600056D RID: 1389 RVA: 0x00019856 File Offset: 0x00017A56
		// (set) Token: 0x0600056E RID: 1390 RVA: 0x0001985E File Offset: 0x00017A5E
		public string ServiceCloudLogin
		{
			get
			{
				return this._serviceCloudLogin;
			}
			set
			{
				if (this._serviceCloudLogin == value)
				{
					return;
				}
				this._serviceCloudLogin = value;
				this.OnPropertyChanged("ServiceCloudLogin");
			}
		}

		// Token: 0x17000189 RID: 393
		// (get) Token: 0x0600056F RID: 1391 RVA: 0x00019881 File Offset: 0x00017A81
		// (set) Token: 0x06000570 RID: 1392 RVA: 0x00019889 File Offset: 0x00017A89
		public string ServiceCloudPassword
		{
			get
			{
				return this._serviceCloudPassword;
			}
			set
			{
				if (this._serviceCloudPassword == value)
				{
					return;
				}
				this._serviceCloudPassword = value;
				this.OnPropertyChanged("ServiceCloudPassword");
			}
		}

		// Token: 0x1700018A RID: 394
		// (get) Token: 0x06000571 RID: 1393 RVA: 0x000198AC File Offset: 0x00017AAC
		// (set) Token: 0x06000572 RID: 1394 RVA: 0x000198B4 File Offset: 0x00017AB4
		public bool DiffWithoutSection
		{
			get
			{
				return this._diffWithoutSection;
			}
			set
			{
				if (this._diffWithoutSection == value)
				{
					return;
				}
				this._diffWithoutSection = value;
				this.OnPropertyChanged("DiffWithoutSection");
			}
		}

		// Token: 0x1700018B RID: 395
		// (get) Token: 0x06000573 RID: 1395 RVA: 0x000198D2 File Offset: 0x00017AD2
		// (set) Token: 0x06000574 RID: 1396 RVA: 0x000198DA File Offset: 0x00017ADA
		public bool DiffWithANSI
		{
			get
			{
				return this._diffWithANSI;
			}
			set
			{
				if (this._diffWithANSI == value)
				{
					return;
				}
				this._diffWithANSI = value;
				this.OnPropertyChanged("DiffWithANSI");
			}
		}

		// Token: 0x1700018C RID: 396
		// (get) Token: 0x06000575 RID: 1397 RVA: 0x000198F8 File Offset: 0x00017AF8
		// (set) Token: 0x06000576 RID: 1398 RVA: 0x00019900 File Offset: 0x00017B00
		public bool ValidateForm
		{
			get
			{
				return this._validateForm;
			}
			set
			{
				this._validateForm = value;
				this.OnPropertyChanged("ValidateForm");
			}
		}

		// Token: 0x1700018D RID: 397
		// (get) Token: 0x06000577 RID: 1399 RVA: 0x00019914 File Offset: 0x00017B14
		// (set) Token: 0x06000578 RID: 1400 RVA: 0x0001991C File Offset: 0x00017B1C
		public bool TabIndexSort
		{
			get
			{
				return this._tabindexsort;
			}
			set
			{
				this._tabindexsort = value;
				this.OnPropertyChanged("TabIndexSort");
			}
		}

		// Token: 0x1700018E RID: 398
		// (get) Token: 0x06000579 RID: 1401 RVA: 0x00019930 File Offset: 0x00017B30
		// (set) Token: 0x0600057A RID: 1402 RVA: 0x00019938 File Offset: 0x00017B38
		public bool LocalizedProperty
		{
			get
			{
				return this._localizedProperty;
			}
			set
			{
				this._localizedProperty = value;
				this.OnPropertyChanged("LocalizedProperty");
			}
		}

		// Token: 0x1700018F RID: 399
		// (get) Token: 0x0600057B RID: 1403 RVA: 0x0001994C File Offset: 0x00017B4C
		// (set) Token: 0x0600057C RID: 1404 RVA: 0x00019954 File Offset: 0x00017B54
		public bool StartShowWelcome
		{
			get
			{
				return this._startShowWelcome;
			}
			set
			{
				if (this._startShowWelcome == value)
				{
					return;
				}
				this._startShowWelcome = value;
				this.OnPropertyChanged("StartShowWelcome");
			}
		}

		// Token: 0x17000190 RID: 400
		// (get) Token: 0x0600057D RID: 1405 RVA: 0x00019972 File Offset: 0x00017B72
		// (set) Token: 0x0600057E RID: 1406 RVA: 0x0001997A File Offset: 0x00017B7A
		public bool DisplayOpenedFiles
		{
			get
			{
				return this._displayOpenedFiles;
			}
			set
			{
				if (this._displayOpenedFiles == value)
				{
					return;
				}
				this._displayOpenedFiles = value;
				this.OnPropertyChanged("DisplayOpenedFiles");
			}
		}

		// Token: 0x17000191 RID: 401
		// (get) Token: 0x0600057F RID: 1407 RVA: 0x00019998 File Offset: 0x00017B98
		// (set) Token: 0x06000580 RID: 1408 RVA: 0x000199A0 File Offset: 0x00017BA0
		public bool ShowExample
		{
			get
			{
				return this._showExample;
			}
			set
			{
				if (this._showExample == value)
				{
					return;
				}
				this._showExample = value;
				this.OnPropertyChanged("ShowExample");
			}
		}

		// Token: 0x17000192 RID: 402
		// (get) Token: 0x06000581 RID: 1409 RVA: 0x000199BE File Offset: 0x00017BBE
		// (set) Token: 0x06000582 RID: 1410 RVA: 0x000199C6 File Offset: 0x00017BC6
		public int MaxRecentFiles
		{
			get
			{
				return this._maxRecentFiles;
			}
			set
			{
				if (this._maxRecentFiles == value)
				{
					return;
				}
				this._maxRecentFiles = value;
				this.OnPropertyChanged("MaxRecentFiles");
			}
		}

		// Token: 0x17000193 RID: 403
		// (get) Token: 0x06000583 RID: 1411 RVA: 0x000199E4 File Offset: 0x00017BE4
		// (set) Token: 0x06000584 RID: 1412 RVA: 0x000199EC File Offset: 0x00017BEC
		public int MaxAmount
		{
			get
			{
				return this._maxAmount;
			}
			set
			{
				if (value > 1000 || value < 0)
				{
					return;
				}
				if (this._maxAmount == value)
				{
					return;
				}
				this._maxAmount = value;
				this.OnPropertyChanged("MaxAmount");
			}
		}

		// Token: 0x17000194 RID: 404
		// (get) Token: 0x06000585 RID: 1413 RVA: 0x00019A17 File Offset: 0x00017C17
		// (set) Token: 0x06000586 RID: 1414 RVA: 0x00019A1F File Offset: 0x00017C1F
		public bool StandardView
		{
			get
			{
				return this._isStandardView;
			}
			set
			{
				if (this._isStandardView == value)
				{
					return;
				}
				this._isStandardView = value;
				this.OnPropertyChanged("StandardView");
			}
		}

		// Token: 0x17000195 RID: 405
		// (get) Token: 0x06000587 RID: 1415 RVA: 0x00019A3D File Offset: 0x00017C3D
		// (set) Token: 0x06000588 RID: 1416 RVA: 0x00019A45 File Offset: 0x00017C45
		public int UndoTimes
		{
			get
			{
				return this._undoTimes;
			}
			set
			{
				if (value > 1000 || value < -1)
				{
					return;
				}
				if (this._undoTimes == value)
				{
					return;
				}
				this._undoTimes = value;
				this.OnPropertyChanged("UndoTimes");
			}
		}

		// Token: 0x17000196 RID: 406
		// (get) Token: 0x06000589 RID: 1417 RVA: 0x00019A70 File Offset: 0x00017C70
		// (set) Token: 0x0600058A RID: 1418 RVA: 0x00019A78 File Offset: 0x00017C78
		public int BackupLimit
		{
			get
			{
				return this._backupLimit;
			}
			set
			{
				if (value > 99 || value < 1)
				{
					return;
				}
				if (this._backupLimit == value)
				{
					return;
				}
				this._backupLimit = value;
				this.OnPropertyChanged("BackupLimit");
			}
		}

		// Token: 0x17000197 RID: 407
		// (get) Token: 0x0600058B RID: 1419 RVA: 0x00019AA0 File Offset: 0x00017CA0
		// (set) Token: 0x0600058C RID: 1420 RVA: 0x00019AA8 File Offset: 0x00017CA8
		public bool RemaindSave
		{
			get
			{
				return this._remaindSave;
			}
			set
			{
				if (this._remaindSave == value)
				{
					return;
				}
				this._remaindSave = value;
				this.OnPropertyChanged("RemaindSave");
			}
		}

		// Token: 0x17000198 RID: 408
		// (get) Token: 0x0600058D RID: 1421 RVA: 0x00019AC6 File Offset: 0x00017CC6
		// (set) Token: 0x0600058E RID: 1422 RVA: 0x00019ACE File Offset: 0x00017CCE
		public bool SimpleDownloadDialog
		{
			get
			{
				return this._simpleDownloadDialog;
			}
			set
			{
				if (this._simpleDownloadDialog == value)
				{
					return;
				}
				this._simpleDownloadDialog = value;
				this.OnPropertyChanged("SimpleDownloadDialog");
			}
		}

		// Token: 0x17000199 RID: 409
		// (get) Token: 0x0600058F RID: 1423 RVA: 0x00019AEC File Offset: 0x00017CEC
		// (set) Token: 0x06000590 RID: 1424 RVA: 0x00019AF4 File Offset: 0x00017CF4
		public bool RemaindDiffSave
		{
			get
			{
				return this._remaindDiffSave;
			}
			set
			{
				if (this._remaindDiffSave == value)
				{
					return;
				}
				this._remaindDiffSave = value;
				this.OnPropertyChanged("RemaindDiffSave");
			}
		}

		// Token: 0x1700019A RID: 410
		// (get) Token: 0x06000591 RID: 1425 RVA: 0x00019B12 File Offset: 0x00017D12
		// (set) Token: 0x06000592 RID: 1426 RVA: 0x00019B1A File Offset: 0x00017D1A
		public bool AlwaysSaveDiff
		{
			get
			{
				return this._alwaysSaveDiff;
			}
			set
			{
				if (this._alwaysSaveDiff == value)
				{
					return;
				}
				this._alwaysSaveDiff = value;
				this.OnPropertyChanged("AlwaysSaveDiff");
			}
		}

		// Token: 0x1700019B RID: 411
		// (get) Token: 0x06000593 RID: 1427 RVA: 0x00019B38 File Offset: 0x00017D38
		// (set) Token: 0x06000594 RID: 1428 RVA: 0x00019B40 File Offset: 0x00017D40
		public string UILang
		{
			get
			{
				return this._UILang;
			}
			set
			{
				if (this._UILang == value)
				{
					return;
				}
				this._UILang = value;
				this.OnPropertyChanged("UILang");
			}
		}

		// Token: 0x1700019C RID: 412
		// (get) Token: 0x06000595 RID: 1429 RVA: 0x00019B63 File Offset: 0x00017D63
		// (set) Token: 0x06000596 RID: 1430 RVA: 0x00019B6B File Offset: 0x00017D6B
		public CommonUsedElement CommonUsedSetting { get; private set; }

		// Token: 0x06000597 RID: 1431 RVA: 0x00019B74 File Offset: 0x00017D74
		public static void CloneSetting(PreferenceModel source, PreferenceModel target)
		{
			target.FontSize = source.FontSize;
			target.FontFamily = source.FontFamily;
			target.TabWidth = source.TabWidth;
			target.WordWrap = source.WordWrap;
			target.CodeCompletion = source.CodeCompletion;
			target.SelfKeyword = source.SelfKeyword;
			target._selfKeywords.Clear();
			target.FavoriteKeyword = source.FavoriteKeyword;
			target._favoriteKeyword.Clear();
			target.Theme = source.Theme;
			target.Background = source.Background;
			target.Foreground = source.Foreground;
			target.UndoTimes = source.UndoTimes;
			target.BackupLimit = source.BackupLimit;
			target.ValidateForm = source.ValidateForm;
			target.TabIndexSort = source.TabIndexSort;
			target.ShowHint = source.ShowHint;
			target.RemaindSave = source.RemaindSave;
			target.SimpleDownloadDialog = source.SimpleDownloadDialog;
			target.RemaindDiffSave = source.RemaindDiffSave;
			target.AlwaysSaveDiff = source.AlwaysSaveDiff;
			target.SimplifyColor = source.SimplifyColor;
			target.HorizontalArrangement = source.HorizontalArrangement;
			target.DiffComment = source.DiffComment;
			target.ViMode = source.ViMode;
			target.TopstdEditPermission = source.TopstdEditPermission;
			target.ServiceCloudLogin = source.ServiceCloudLogin;
			target.ServiceCloudPassword = source.ServiceCloudPassword;
			target.UILang = source.UILang;
			target.MaxRecentFiles = source.MaxRecentFiles;
			target.DiffWithoutSection = source.DiffWithoutSection;
			target.DiffWithANSI = source.DiffWithANSI;
			target.StartShowWelcome = source.StartShowWelcome;
			target.DisplayOpenedFiles = source.DisplayOpenedFiles;
			target.ShowExample = source.ShowExample;
			target.MaxAmount = source.MaxAmount;
			target.LocalizedProperty = source.LocalizedProperty;
		}

		// Token: 0x040001F9 RID: 505
		private int _fontSize = 12;

		// Token: 0x040001FA RID: 506
		private FontFamily _fontFamily = new FontFamily("Consolas");

		// Token: 0x040001FB RID: 507
		private int _tabWidth = 3;

		// Token: 0x040001FC RID: 508
		private bool _wordWrap;

		// Token: 0x040001FD RID: 509
		private bool _codeCompletion;

		// Token: 0x040001FE RID: 510
		private List<string> _selfKeywords;

		// Token: 0x040001FF RID: 511
		private string _selfKeywordSource = string.Empty;

		// Token: 0x04000200 RID: 512
		private List<string> _favoriteKeyword;

		// Token: 0x04000201 RID: 513
		private string _favoriteKeywordSource = string.Empty;

		// Token: 0x04000202 RID: 514
		private ThemeOptions _theme;

		// Token: 0x04000203 RID: 515
		private readonly string COLOR_CHECK = "^[a-fA-F0-9]+$";

		// Token: 0x04000204 RID: 516
		private string _background = "FFFFFF";

		// Token: 0x04000205 RID: 517
		private string _foreground = "000000";

		// Token: 0x04000206 RID: 518
		private bool _showHint;

		// Token: 0x04000207 RID: 519
		private bool _simplifyColor;

		// Token: 0x04000208 RID: 520
		private bool _horizontalArrangement;

		// Token: 0x04000209 RID: 521
		private bool _diffComment;

		// Token: 0x0400020A RID: 522
		private bool _topstdEditPermission;

		// Token: 0x0400020B RID: 523
		private bool _viMode;

		// Token: 0x0400020C RID: 524
		private string _serviceCloudLogin = string.Empty;

		// Token: 0x0400020D RID: 525
		private string _serviceCloudPassword = string.Empty;

		// Token: 0x0400020E RID: 526
		private bool _diffWithoutSection;

		// Token: 0x0400020F RID: 527
		private bool _diffWithANSI;

		// Token: 0x04000210 RID: 528
		private bool _validateForm = true;

		// Token: 0x04000211 RID: 529
		private bool _tabindexsort = true;

		// Token: 0x04000212 RID: 530
		private bool _localizedProperty;

		// Token: 0x04000213 RID: 531
		private bool _startShowWelcome;

		// Token: 0x04000214 RID: 532
		private bool _displayOpenedFiles;

		// Token: 0x04000215 RID: 533
		private bool _showExample;

		// Token: 0x04000216 RID: 534
		private int _maxRecentFiles = 10;

		// Token: 0x04000217 RID: 535
		private int _maxAmount = 10;

		// Token: 0x04000218 RID: 536
		private bool _isStandardView = true;

		// Token: 0x04000219 RID: 537
		private int _undoTimes = -1;

		// Token: 0x0400021A RID: 538
		private int _backupLimit = 1;

		// Token: 0x0400021B RID: 539
		private bool _remaindSave = true;

		// Token: 0x0400021C RID: 540
		private bool _simpleDownloadDialog;

		// Token: 0x0400021D RID: 541
		private bool _remaindDiffSave = true;

		// Token: 0x0400021E RID: 542
		private bool _alwaysSaveDiff = true;

		// Token: 0x0400021F RID: 543
		private string _UILang = CultureInfo.CurrentUICulture.Name;
	}
}
