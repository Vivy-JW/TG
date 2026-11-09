using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
using System.Windows;
using System.Xml.Linq;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Exceptions;
using SpecDesignerCommon.ViewModel;
using SpecDesignerCustomException;
using SpecDesignerPreference;
using UndoRedoFramework;

namespace SpecDesignerCommon
{
	// Token: 0x02000110 RID: 272
	public class TzpManager
	{
		// Token: 0x14000022 RID: 34
		// (add) Token: 0x06000967 RID: 2407 RVA: 0x0002FFA4 File Offset: 0x0002E1A4
		// (remove) Token: 0x06000968 RID: 2408 RVA: 0x0002FFDC File Offset: 0x0002E1DC
		public event TzpManager.TzpSavedHandler TzpSaved;

		// Token: 0x1700027D RID: 637
		// (get) Token: 0x06000969 RID: 2409 RVA: 0x00030011 File Offset: 0x0002E211
		// (set) Token: 0x0600096A RID: 2410 RVA: 0x00030019 File Offset: 0x0002E219
		public string ZipFile { get; set; }

		// Token: 0x1700027E RID: 638
		// (get) Token: 0x0600096B RID: 2411 RVA: 0x00030022 File Offset: 0x0002E222
		// (set) Token: 0x0600096C RID: 2412 RVA: 0x0003002A File Offset: 0x0002E22A
		public string CitedZipFile { get; set; }

		// Token: 0x1700027F RID: 639
		// (get) Token: 0x0600096D RID: 2413 RVA: 0x00030033 File Offset: 0x0002E233
		// (set) Token: 0x0600096E RID: 2414 RVA: 0x0003003B File Offset: 0x0002E23B
		public string ProgramName { get; set; }

		// Token: 0x17000280 RID: 640
		// (get) Token: 0x0600096F RID: 2415 RVA: 0x00030044 File Offset: 0x0002E244
		// (set) Token: 0x06000970 RID: 2416 RVA: 0x0003004C File Offset: 0x0002E24C
		public string StdProgramName { get; set; }

		// Token: 0x17000281 RID: 641
		// (get) Token: 0x06000971 RID: 2417 RVA: 0x00030055 File Offset: 0x0002E255
		// (set) Token: 0x06000972 RID: 2418 RVA: 0x0003005D File Offset: 0x0002E25D
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x17000282 RID: 642
		// (get) Token: 0x06000973 RID: 2419 RVA: 0x00030066 File Offset: 0x0002E266
		// (set) Token: 0x06000974 RID: 2420 RVA: 0x0003006E File Offset: 0x0002E26E
		public string ModuleName { get; set; }

		// Token: 0x17000283 RID: 643
		// (get) Token: 0x06000975 RID: 2421 RVA: 0x00030078 File Offset: 0x0002E278
		public bool Booking
		{
			get
			{
				XElement xelement = null;
				switch (this.Type)
				{
				case TzpType.ReportSpec:
					xelement = XElement.Parse(this.RSD);
					break;
				case TzpType.ReportCode:
				case TzpType.Code:
					xelement = XElement.Parse(this.TAP);
					break;
				case TzpType.CodeSpec:
					xelement = XElement.Parse(this.CSD);
					break;
				case TzpType.Form:
					xelement = XElement.Parse(this.Tsd);
					break;
				}
				return xelement != null && (xelement.Attribute("booking") != null && xelement.Attribute("booking").Value.Equals("y", StringComparison.CurrentCultureIgnoreCase));
			}
		}

		// Token: 0x17000284 RID: 644
		// (get) Token: 0x06000976 RID: 2422 RVA: 0x00030120 File Offset: 0x0002E320
		public bool CodeBooking
		{
			get
			{
				XElement xelement = null;
				TzpType type = this.Type;
				if (type == TzpType.Form)
				{
					xelement = XElement.Parse(this.Tsd);
				}
				return xelement != null && (xelement.Attribute("code_booking") != null && xelement.Attribute("code_booking").Value.Equals("y", StringComparison.CurrentCultureIgnoreCase));
			}
		}

		// Token: 0x17000285 RID: 645
		// (get) Token: 0x06000977 RID: 2423 RVA: 0x00030184 File Offset: 0x0002E384
		public bool CanPrecompile
		{
			get
			{
				XElement xelement = null;
				switch (this.Type)
				{
				case TzpType.Code:
					xelement = XElement.Parse(this.TAP);
					break;
				case TzpType.Form:
					xelement = XElement.Parse(this.Tsd);
					break;
				}
				return xelement != null && xelement.Attribute("pre_compile") != null && xelement.Attribute("pre_compile").Value.Equals("y", StringComparison.CurrentCultureIgnoreCase);
			}
		}

		// Token: 0x17000286 RID: 646
		// (get) Token: 0x06000978 RID: 2424 RVA: 0x00030204 File Offset: 0x0002E404
		public bool IsDiff
		{
			get
			{
				return this._isDiff;
			}
		}

		// Token: 0x17000287 RID: 647
		// (get) Token: 0x06000979 RID: 2425 RVA: 0x0003020C File Offset: 0x0002E40C
		public bool IsMajorAbnormal
		{
			get
			{
				return this._isMajorAbnormal;
			}
		}

		// Token: 0x17000288 RID: 648
		// (get) Token: 0x0600097A RID: 2426 RVA: 0x00030214 File Offset: 0x0002E414
		public bool isIndFun
		{
			get
			{
				return this._isIndFun;
			}
		}

		// Token: 0x17000289 RID: 649
		// (get) Token: 0x0600097B RID: 2427 RVA: 0x0003021C File Offset: 0x0002E41C
		public TzpType Type
		{
			get
			{
				string extension = Path.GetExtension(this.ZipFile);
				string text;
				switch (text = extension.ToLower())
				{
				case ".tzs":
					return TzpType.Form;
				case ".tzc":
					return TzpType.Code;
				case ".tzd":
					return TzpType.CodeSpec;
				case ".tzr":
					return TzpType.ReportSpec;
				case ".tzg":
					return TzpType.ReportCode;
				case ".tzt":
					return TzpType.Report;
				case ".tzx":
					this._isDiff = true;
					return TzpType.Code;
				case ".tzv":
					this.IsSimpleForm = true;
					return TzpType.Form;
				case ".tzf":
					this._isIndFun = true;
					return TzpType.Code;
				}
				return TzpType.None;
			}
		}

		// Token: 0x1700028A RID: 650
		// (get) Token: 0x0600097C RID: 2428 RVA: 0x0003032A File Offset: 0x0002E52A
		// (set) Token: 0x0600097D RID: 2429 RVA: 0x00030332 File Offset: 0x0002E532
		public bool IsStandardProgram { get; private set; }

		// Token: 0x1700028B RID: 651
		// (get) Token: 0x0600097E RID: 2430 RVA: 0x0003033B File Offset: 0x0002E53B
		// (set) Token: 0x0600097F RID: 2431 RVA: 0x00030343 File Offset: 0x0002E543
		public bool IsNormalStyle { get; private set; }

		// Token: 0x1700028C RID: 652
		// (get) Token: 0x06000980 RID: 2432 RVA: 0x0003034C File Offset: 0x0002E54C
		// (set) Token: 0x06000981 RID: 2433 RVA: 0x00030354 File Offset: 0x0002E554
		public string ProgIdentity { get; private set; }

		// Token: 0x1700028D RID: 653
		// (get) Token: 0x06000982 RID: 2434 RVA: 0x0003035D File Offset: 0x0002E55D
		// (set) Token: 0x06000983 RID: 2435 RVA: 0x00030365 File Offset: 0x0002E565
		public string ProgVer { get; private set; }

		// Token: 0x06000984 RID: 2436 RVA: 0x0003036E File Offset: 0x0002E56E
		public void SetFreeStyle()
		{
			this.IsNormalStyle = false;
		}

		// Token: 0x1700028E RID: 654
		// (get) Token: 0x06000985 RID: 2437 RVA: 0x00030377 File Offset: 0x0002E577
		// (set) Token: 0x06000986 RID: 2438 RVA: 0x0003037F File Offset: 0x0002E57F
		public string ProgType { get; private set; }

		// Token: 0x06000987 RID: 2439 RVA: 0x00030388 File Offset: 0x0002E588
		public TzpManager(string zipFile)
		{
			this.ZipFile = zipFile;
			if (this.CheckReleaseVersion(zipFile) && this.InCurrentWorkspace(zipFile))
			{
				this.initReadingFlags();
				PackageManager.Unpacking(zipFile, this);
				this.LoadProgramInfomation();
			}
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.TzpFileClosed));
		}

		// Token: 0x06000988 RID: 2440 RVA: 0x000303F9 File Offset: 0x0002E5F9
		private PackageKey CreateKey(string program, TzpType packType)
		{
			if (string.IsNullOrWhiteSpace(program) || packType == TzpType.None)
			{
				throw new NullReferenceException("TzpManager");
			}
			return new PackageKey(program, packType);
		}

		// Token: 0x06000989 RID: 2441 RVA: 0x00030418 File Offset: 0x0002E618
		private bool InCurrentWorkspace(string zipFile)
		{
			string directoryName = Path.GetDirectoryName(zipFile);
			if (SettingManager.Get().CurrentSetting.Connection.InWorkspace(directoryName))
			{
				return true;
			}
			string text = Application.Current.FindResource("Message_TzpNotInWorkspace") as string;
			throw new NotInCurrentWorkspaceException(string.Format(text, zipFile));
		}

		// Token: 0x0600098A RID: 2442 RVA: 0x00030468 File Offset: 0x0002E668
		private void PackChecking()
		{
			using (Dictionary<string, bool?>.ValueCollection.Enumerator enumerator = this._readingFlags.Values.GetEnumerator())
			{
				while (enumerator.MoveNext())
				{
					if (enumerator.Current == false)
					{
						return;
					}
				}
			}
			if (PackageManager.Packing(this.ZipFile, this))
			{
				if (this.TzpSaved != null)
				{
					this.TzpSaved(this, new TzpSavedEventArgs(this.ProgramKey, this.Type));
				}
				this.initReadingFlags();
				GC.Collect();
			}
		}

		// Token: 0x0600098B RID: 2443 RVA: 0x00030510 File Offset: 0x0002E710
		private void initReadingFlags()
		{
			List<string> list = this._readingFlags.Keys.ToList<string>();
			foreach (string text in list)
			{
				if (this._readingFlags[text] != null)
				{
					this._readingFlags[text] = new bool?(false);
				}
			}
		}

		// Token: 0x0600098C RID: 2444 RVA: 0x00030590 File Offset: 0x0002E790
		private void LoadProgramInfomation()
		{
			XElement xelement;
			try
			{
				string text = string.Empty;
				switch (this.Type)
				{
				case TzpType.ReportSpec:
					text = this.RSD;
					goto IL_006D;
				case TzpType.ReportCode:
				case TzpType.Code:
					text = this.TAP;
					goto IL_006D;
				case TzpType.CodeSpec:
					text = this.CSD;
					goto IL_006D;
				case TzpType.Form:
					text = this.Tsd;
					goto IL_006D;
				}
				throw new FileLoadException(Application.Current.FindResource("Message_UnknownFileType") as string);
				IL_006D:
				xelement = XElement.Parse(text);
				this.IsStandardProgram = xelement.Attribute("prog").Value.Equals(xelement.Attribute("std_prog").Value);
				this.IsNormalStyle = !this.IsFreeStyle(xelement);
				this.ModuleName = xelement.Attribute("module").Value;
				this.ProgramName = xelement.Attribute("prog").Value;
				this.ProgramKey = this.CreateKey(this.ProgramName, this.Type);
				this.ProgType = xelement.Attribute("type").Value;
				this.ProgIdentity = xelement.Attribute("identity").Value;
				this.StdProgramName = xelement.Attribute("std_prog").Value;
				this.ProgVer = xelement.Attribute("ver").Value;
			}
			catch
			{
				throw new FileFormatErrorException(string.Format("{0}:{1}", this.ZipFile, Application.Current.FindResource("Message_NecessaryFileMssing") as string));
			}
			if (!this.IsStandardProgram)
			{
				string text2 = "tmp";
				TzpType type = this.Type;
				if (type != TzpType.Report)
				{
					switch (type)
					{
					case TzpType.Code:
						text2 = "tmc";
						break;
					case TzpType.Form:
						text2 = "tms";
						break;
					}
				}
				else
				{
					text2 = "tmg";
				}
				this.CitedZipFile = string.Format("{0}\\{1}.{2}", Path.GetDirectoryName(this.ZipFile), xelement.Attribute("std_prog").Value, text2);
				if (!File.Exists(this.CitedZipFile))
				{
					throw new FileNotFoundException(string.Format(Application.Current.FindResource("Message_CitedFileMssing") as string, this.CitedZipFile));
				}
				PackageManager.UnpackingCited(this.CitedZipFile, this);
			}
		}

		// Token: 0x0600098D RID: 2445 RVA: 0x0003081C File Offset: 0x0002EA1C
		private bool IsFreeStyle(XElement root)
		{
			XElement xelement = root.Element("other");
			return xelement != null && xelement.Element("free_style").Attribute("value").Value == "Y";
		}

		// Token: 0x0600098E RID: 2446 RVA: 0x00030872 File Offset: 0x0002EA72
		internal void LoadAddPoint(string content)
		{
			XElement.Parse(content);
			this.TAP = content;
			this._readingFlags["tap"] = new bool?(false);
		}

		// Token: 0x0600098F RID: 2447 RVA: 0x00030898 File Offset: 0x0002EA98
		internal void LoadUpdatedAddPoint(string content)
		{
			this._readingFlags["tap2"] = new bool?(false);
		}

		// Token: 0x06000990 RID: 2448 RVA: 0x000308B0 File Offset: 0x0002EAB0
		public void Load4glFile(string content)
		{
			this._readingFlags["4gl"] = new bool?(false);
		}

		// Token: 0x1700028F RID: 655
		// (get) Token: 0x06000991 RID: 2449 RVA: 0x000308C8 File Offset: 0x0002EAC8
		// (set) Token: 0x06000992 RID: 2450 RVA: 0x000308D0 File Offset: 0x0002EAD0
		public XElement CitedAddPoint { get; private set; }

		// Token: 0x06000993 RID: 2451 RVA: 0x000308D9 File Offset: 0x0002EAD9
		public void LoadCitedAddPoint(string content)
		{
			this.CitedAddPoint = XElement.Parse(content);
		}

		// Token: 0x06000994 RID: 2452 RVA: 0x000308E8 File Offset: 0x0002EAE8
		private bool CheckReleaseVersion(string tzp)
		{
			Version version = SettingManager.Get().Version;
			Version version2 = new Version(PackageManager.SeekReleaseVersion(tzp));
			if (version2 != null && (version2.Major != version.Major || version2.Minor != version.Minor))
			{
				throw new VersionIncompatibleException(version, version2);
			}
			return true;
		}

		// Token: 0x06000995 RID: 2453 RVA: 0x0003093A File Offset: 0x0002EB3A
		public void SaveAddPoint(string content)
		{
			this.TAP = content;
			this._readingFlags["tap"] = new bool?(true);
			this.PackChecking();
		}

		// Token: 0x06000996 RID: 2454 RVA: 0x0003095F File Offset: 0x0002EB5F
		public void SaveUpdatedAddPoint(string content)
		{
			if (this._readingFlags.ContainsKey("tap2"))
			{
				this.UpdatedTAP = content;
				this._readingFlags["tap2"] = new bool?(true);
				this.PackChecking();
			}
		}

		// Token: 0x06000997 RID: 2455 RVA: 0x00030996 File Offset: 0x0002EB96
		public void SaveFullCode(string content)
		{
			this.FullCode = content;
			this._readingFlags["4gl"] = new bool?(true);
			this.PackChecking();
		}

		// Token: 0x06000998 RID: 2456 RVA: 0x000309BB File Offset: 0x0002EBBB
		public void SaveDiffSource()
		{
			this._readingFlags["src"] = new bool?(true);
			this.PackChecking();
		}

		// Token: 0x06000999 RID: 2457 RVA: 0x000309D9 File Offset: 0x0002EBD9
		public void SaveDiffSource(string content)
		{
			this.DIFF_SRC = content;
			this.SaveDiffSource();
		}

		// Token: 0x17000290 RID: 656
		// (get) Token: 0x0600099A RID: 2458 RVA: 0x000309E8 File Offset: 0x0002EBE8
		// (set) Token: 0x0600099B RID: 2459 RVA: 0x000309F0 File Offset: 0x0002EBF0
		public string TGL { get; private set; }

		// Token: 0x0600099C RID: 2460 RVA: 0x000309F9 File Offset: 0x0002EBF9
		public void LoadCodeFile(string content)
		{
			this.TGL = content.TrimEnd(new char[0]);
			this._readingFlags["tgl"] = new bool?(false);
		}

		// Token: 0x0600099D RID: 2461 RVA: 0x00030A23 File Offset: 0x0002EC23
		public void SaveCodeFile(string content)
		{
			this.TGL = content;
			this._readingFlags["tgl"] = new bool?(true);
			this.PackChecking();
		}

		// Token: 0x17000291 RID: 657
		// (get) Token: 0x0600099E RID: 2462 RVA: 0x00030A48 File Offset: 0x0002EC48
		// (set) Token: 0x0600099F RID: 2463 RVA: 0x00030A50 File Offset: 0x0002EC50
		public string GeneroFormString { get; private set; }

		// Token: 0x060009A0 RID: 2464 RVA: 0x00030A59 File Offset: 0x0002EC59
		public void LoadFormFile(string content)
		{
			this.GeneroFormString = content;
			this._readingFlags["4fd"] = new bool?(false);
		}

		// Token: 0x060009A1 RID: 2465 RVA: 0x00030A78 File Offset: 0x0002EC78
		public void SaveFormFile(string content)
		{
			this.GeneroFormString = "<?xml version='1.0' encoding='UTF-8'?>\n" + content;
			this._readingFlags["4fd"] = new bool?(true);
			this.PackChecking();
		}

		// Token: 0x17000292 RID: 658
		// (get) Token: 0x060009A2 RID: 2466 RVA: 0x00030AA7 File Offset: 0x0002ECA7
		// (set) Token: 0x060009A3 RID: 2467 RVA: 0x00030AAF File Offset: 0x0002ECAF
		public string FormLocalizedStrings { get; private set; }

		// Token: 0x060009A4 RID: 2468 RVA: 0x00030AB8 File Offset: 0x0002ECB8
		public string GetLocalizedString(string name)
		{
			if (this.LocalStringTable.ContainsKey(name))
			{
				return this.LocalStringTable[name];
			}
			return null;
		}

		// Token: 0x060009A5 RID: 2469 RVA: 0x00030AD8 File Offset: 0x0002ECD8
		public string GetLocalizedStringWithEnd(string newValue)
		{
			foreach (KeyValuePair<string, string> keyValuePair in this.LocalStringTable)
			{
				if (keyValuePair.Key.EndsWith(newValue))
				{
					return keyValuePair.Value;
				}
			}
			return null;
		}

		// Token: 0x060009A6 RID: 2470 RVA: 0x00030B40 File Offset: 0x0002ED40
		public void UpdateOrCreateLocalizedString(string text, string oldKey, string newKey)
		{
			if (this.LocalStringTable.ContainsKey(oldKey))
			{
				this.LocalStringTable.Remove(oldKey);
			}
			if (this.LocalStringTable.ContainsKey(newKey))
			{
				this.LocalStringTable.Remove(newKey);
			}
			this.LocalStringTable.Add(newKey, text);
		}

		// Token: 0x060009A7 RID: 2471 RVA: 0x00030B90 File Offset: 0x0002ED90
		public void LoadFormLocalizedStrings(string content)
		{
			this.FormLocalizedStrings = content;
			StringReader stringReader = new StringReader(content);
			this.LocalStringTable.Clear();
			string text;
			while ((text = stringReader.ReadLine()) != null)
			{
				if (!string.IsNullOrEmpty(text))
				{
					string[] array = text.Split(new char[] { '=' });
					string text2 = array[0].Trim();
					string text3 = array[1].Trim();
					text2 = text2.Substring(1, text2.Length - 2);
					text3 = text3.Substring(1, text3.Length - 2);
					if (this.LocalStringTable.ContainsKey(text2))
					{
						this.LocalStringTable[text2] = text3;
					}
					else
					{
						this.LocalStringTable.Add(text2, text3);
					}
				}
			}
			this._readingFlags["str"] = new bool?(true);
		}

		// Token: 0x060009A8 RID: 2472 RVA: 0x00030C64 File Offset: 0x0002EE64
		public void SaveFormLocalizedStrings()
		{
			StringBuilder stringBuilder = new StringBuilder();
			foreach (KeyValuePair<string, string> keyValuePair in this.LocalStringTable)
			{
				string text = string.Concat(new string[] { "\"", keyValuePair.Key, "\" = \"", keyValuePair.Value, "\"" });
				stringBuilder.AppendLine(text);
			}
			this.FormLocalizedStrings = stringBuilder.ToString();
			this._readingFlags["str"] = new bool?(true);
			this.PackChecking();
		}

		// Token: 0x17000293 RID: 659
		// (get) Token: 0x060009A9 RID: 2473 RVA: 0x00030D28 File Offset: 0x0002EF28
		// (set) Token: 0x060009AA RID: 2474 RVA: 0x00030D30 File Offset: 0x0002EF30
		public SpecificationInfo SpecificationInfo { get; private set; }

		// Token: 0x060009AB RID: 2475 RVA: 0x00030D39 File Offset: 0x0002EF39
		public void LoadSpecificationInfo(string content)
		{
			this.Tsd = content;
			this._readingFlags["tsd"] = new bool?(false);
		}

		// Token: 0x060009AC RID: 2476 RVA: 0x00030D58 File Offset: 0x0002EF58
		public void LoadUpdatedSpecificationInfo(string content)
		{
			this._readingFlags["tsd2"] = new bool?(false);
		}

		// Token: 0x060009AD RID: 2477 RVA: 0x00030D70 File Offset: 0x0002EF70
		public void LoadCiteTsd(string content)
		{
			if (content != null)
			{
				this.CiteTsd = content;
			}
		}

		// Token: 0x060009AE RID: 2478 RVA: 0x00030D7C File Offset: 0x0002EF7C
		public void SaveSpecificationInfo(string content)
		{
			this._readingFlags["tsd"] = new bool?(true);
			if (!this.IsSimpleForm)
			{
				this.Tsd = content;
			}
			this.PackChecking();
		}

		// Token: 0x060009AF RID: 2479 RVA: 0x00030DA9 File Offset: 0x0002EFA9
		public void SaveUpdatedSpecificationInfo(string content)
		{
			if (this._readingFlags.ContainsKey("tsd2"))
			{
				this._readingFlags["tsd2"] = new bool?(true);
				this.UpdatedTSD = content;
				this.PackChecking();
			}
		}

		// Token: 0x060009B0 RID: 2480 RVA: 0x00030DE0 File Offset: 0x0002EFE0
		public void LoadFormDiffList(string content)
		{
			try
			{
				this.FormDiffList = XElement.Parse(content);
			}
			catch
			{
				this.FormDiffList = null;
			}
		}

		// Token: 0x060009B1 RID: 2481 RVA: 0x00030E18 File Offset: 0x0002F018
		public void LoadActionDefaults(string content)
		{
			this.TiptopActionDefaults = content;
		}

		// Token: 0x060009B2 RID: 2482 RVA: 0x00030E21 File Offset: 0x0002F021
		public void LoadCustomActionDefaults(string content)
		{
			this.CustomActionDefaults = content;
		}

		// Token: 0x060009B3 RID: 2483 RVA: 0x00030E2C File Offset: 0x0002F02C
		private void MergeToActionDefaults(string content)
		{
			if (content == null)
			{
				return;
			}
			XElement xelement = XElement.Parse(content, LoadOptions.None);
			if (this.ActionDefaults == null)
			{
				this.ActionDefaults = xelement;
				return;
			}
			if (xelement.Elements("ActionDefault").Count<XElement>() > 0)
			{
				this.ActionDefaults.Add(xelement.Elements("ActionDefault"));
			}
		}

		// Token: 0x17000294 RID: 660
		// (get) Token: 0x060009B4 RID: 2484 RVA: 0x00030E88 File Offset: 0x0002F088
		// (set) Token: 0x060009B5 RID: 2485 RVA: 0x00030E90 File Offset: 0x0002F090
		public string CSD { get; private set; }

		// Token: 0x17000295 RID: 661
		// (get) Token: 0x060009B6 RID: 2486 RVA: 0x00030E99 File Offset: 0x0002F099
		// (set) Token: 0x060009B7 RID: 2487 RVA: 0x00030EA1 File Offset: 0x0002F0A1
		public FglSpecification FglSpecificationInfo { get; private set; }

		// Token: 0x060009B8 RID: 2488 RVA: 0x00030EAA File Offset: 0x0002F0AA
		public void LoadSpecificationForCode(string content)
		{
			this.CSD = content;
			this._readingFlags["csd"] = new bool?(false);
		}

		// Token: 0x060009B9 RID: 2489 RVA: 0x00030EC9 File Offset: 0x0002F0C9
		public void SaveSpecificationForCode()
		{
			this.CSD = this.FglSpecificationInfo.ToXElement().ToString();
			this._readingFlags["csd"] = new bool?(true);
			this.PackChecking();
		}

		// Token: 0x17000296 RID: 662
		// (get) Token: 0x060009BA RID: 2490 RVA: 0x00030EFD File Offset: 0x0002F0FD
		// (set) Token: 0x060009BB RID: 2491 RVA: 0x00030F05 File Offset: 0x0002F105
		public string RSD { get; private set; }

		// Token: 0x17000297 RID: 663
		// (get) Token: 0x060009BC RID: 2492 RVA: 0x00030F0E File Offset: 0x0002F10E
		// (set) Token: 0x060009BD RID: 2493 RVA: 0x00030F16 File Offset: 0x0002F116
		public ReportSpecification ReportSpecificationInfo { get; private set; }

		// Token: 0x060009BE RID: 2494 RVA: 0x00030F1F File Offset: 0x0002F11F
		public void LoadSpecificationForReport(string content)
		{
			this.RSD = content;
			this._readingFlags["rsd"] = new bool?(false);
		}

		// Token: 0x060009BF RID: 2495 RVA: 0x00030F3E File Offset: 0x0002F13E
		public void SaveSpecificationForReport()
		{
			this.RSD = this.ReportSpecificationInfo.ToXElement().ToString();
			this._readingFlags["rsd"] = new bool?(true);
			this.PackChecking();
		}

		// Token: 0x17000298 RID: 664
		// (get) Token: 0x060009C0 RID: 2496 RVA: 0x00030F72 File Offset: 0x0002F172
		// (set) Token: 0x060009C1 RID: 2497 RVA: 0x00030F7A File Offset: 0x0002F17A
		public string DIFF_SRC { get; private set; }

		// Token: 0x17000299 RID: 665
		// (get) Token: 0x060009C2 RID: 2498 RVA: 0x00030F83 File Offset: 0x0002F183
		// (set) Token: 0x060009C3 RID: 2499 RVA: 0x00030F8B File Offset: 0x0002F18B
		public XElement DIFF_TAP { get; private set; }

		// Token: 0x060009C4 RID: 2500 RVA: 0x00030F94 File Offset: 0x0002F194
		public void LoadDiffSrcFile(string content)
		{
			this._readingFlags["src"] = new bool?(false);
			this.DIFF_SRC = content.TrimEnd(new char[0]);
		}

		// Token: 0x060009C5 RID: 2501 RVA: 0x00030FC0 File Offset: 0x0002F1C0
		public void LoadDiffSrcTAPFile(string content)
		{
			try
			{
				this.DIFF_TAP = XElement.Parse(content.TrimEnd(new char[0]));
			}
			catch
			{
				this.DIFF_TAP = null;
			}
		}

		// Token: 0x060009C6 RID: 2502 RVA: 0x00031000 File Offset: 0x0002F200
		public void LoadADT(string content)
		{
			XElement.Parse(content);
			this.ADT = content;
		}

		// Token: 0x060009C7 RID: 2503 RVA: 0x00031010 File Offset: 0x0002F210
		public void LoadinfoXML(string content)
		{
			XElement xelement = XElement.Parse(content);
			if (!string.IsNullOrEmpty(xelement.Attribute("cond").Value))
			{
				this.infoXML = xelement.Attribute("cond").Value.Substring(0, xelement.Attribute("cond").Value.Length - 1);
			}
			this._isMajorAbnormal = true;
		}

		// Token: 0x1700029A RID: 666
		// (get) Token: 0x060009C8 RID: 2504 RVA: 0x00031084 File Offset: 0x0002F284
		// (set) Token: 0x060009C9 RID: 2505 RVA: 0x0003108B File Offset: 0x0002F28B
		public static TzpManager Current { get; set; }

		// Token: 0x1700029B RID: 667
		// (get) Token: 0x060009CA RID: 2506 RVA: 0x00031093 File Offset: 0x0002F293
		// (set) Token: 0x060009CB RID: 2507 RVA: 0x0003109B File Offset: 0x0002F29B
		public bool IsSimpleForm { get; private set; }

		// Token: 0x1700029C RID: 668
		// (get) Token: 0x060009CC RID: 2508 RVA: 0x000310A4 File Offset: 0x0002F2A4
		// (set) Token: 0x060009CD RID: 2509 RVA: 0x000310AC File Offset: 0x0002F2AC
		public string RefFormString { get; private set; }

		// Token: 0x060009CE RID: 2510 RVA: 0x000310B5 File Offset: 0x0002F2B5
		public void LoadRefFormFile(string content)
		{
			this.RefFormString = content;
		}

		// Token: 0x060009CF RID: 2511 RVA: 0x0003111C File Offset: 0x0002F31C
		public bool IsSimpleRefFormHidden(string name)
		{
			bool? flag = null;
			if (this.IsSimpleForm)
			{
				this.RefFormNode = XElement.Parse(this.RefFormString).Element("Form");
				if (this.RefFormNode != null)
				{
					IEnumerable<string> enumerable = from el in this.RefFormNode.Descendants()
						where (string)el.Attribute("name") == name && el.Attribute("hidden") != null
						select el.Attribute("hidden").Value;
					if (enumerable.Count<string>() > 0 && !string.IsNullOrEmpty(enumerable.First<string>()))
					{
						flag = new bool?(bool.Parse(enumerable.First<string>()));
					}
				}
			}
			return flag != true;
		}

		// Token: 0x060009D0 RID: 2512 RVA: 0x000311F8 File Offset: 0x0002F3F8
		internal void LoadTzpFinished()
		{
			TzpType type = this.Type;
			if (type == TzpType.Form)
			{
				this.MergeToActionDefaults(this.TiptopActionDefaults);
			}
			try
			{
				this.ReportSpecificationInfo = ReportSpecification.Parse(this.RSD, this);
				this.FglSpecificationInfo = FglSpecification.Parse(this.CSD, this);
				this.SpecificationInfo = SpecificationInfo.Create(this);
			}
			catch (FileNotFoundException ex)
			{
				EventAggregatorManager.Global.GetEvent<TzpFileClose>().Publish(this.ProgramKey);
				throw new FileFormatErrorException(string.Format(Application.Current.FindResource("Message_FileMssing") as string, ex.Message));
			}
			catch (Exception)
			{
				EventAggregatorManager.Global.GetEvent<TzpFileClose>().Publish(this.ProgramKey);
				throw new FileFormatErrorException(Application.Current.FindResource("Message_FileContentError") as string);
			}
			try
			{
				if (this.CiteTsd != null && this.Type == TzpType.Form)
				{
					this.SpecificationInfo.SetCitedSTD(this.CiteTsd);
				}
			}
			catch
			{
				EventAggregatorManager.Global.GetEvent<TzpFileClose>().Publish(this.ProgramKey);
				throw new FileFormatErrorException(Application.Current.FindResource("Message_CitedFileContentError") as string);
			}
			TzpType type2 = this.Type;
			if (type2 != TzpType.Form)
			{
				return;
			}
			SettingManager.Get().undoRedoManagerMap.Add(this.ProgramKey, new UndoRedoManager(PreferenceManager.Current.Settings.UndoTimes));
		}

		// Token: 0x060009D1 RID: 2513 RVA: 0x0003136C File Offset: 0x0002F56C
		public void TzpFileClosed(PackageKey key)
		{
			if (key != this.ProgramKey)
			{
				return;
			}
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Unsubscribe(new Action<PackageKey>(this.TzpFileClosed));
			this._readingFlags.Clear();
			this._readingFlags = null;
			this.CitedAddPoint = null;
			this.LocalStringTable.Clear();
			this.LocalStringTable = null;
			this.SpecificationInfo = null;
			this.ActionDefaults = null;
			this.FglSpecificationInfo = null;
		}

		// Token: 0x060009D2 RID: 2514 RVA: 0x000313E3 File Offset: 0x0002F5E3
		public void LoadBindings(string content)
		{
			this.ElementBindings = content;
		}

		// Token: 0x04000383 RID: 899
		private const string XmlDeclaration = "<?xml version='1.0' encoding='UTF-8'?>\n";

		// Token: 0x04000385 RID: 901
		private bool _isDiff;

		// Token: 0x04000386 RID: 902
		private bool _isMajorAbnormal;

		// Token: 0x04000387 RID: 903
		private bool _isIndFun;

		// Token: 0x04000388 RID: 904
		private Dictionary<string, bool?> _readingFlags = new Dictionary<string, bool?>();

		// Token: 0x04000389 RID: 905
		public string TAP;

		// Token: 0x0400038A RID: 906
		public string CiteTAP;

		// Token: 0x0400038B RID: 907
		public string FullCode;

		// Token: 0x0400038C RID: 908
		public string UpdatedTAP;

		// Token: 0x0400038D RID: 909
		public Dictionary<string, string> LocalStringTable = new Dictionary<string, string>();

		// Token: 0x0400038E RID: 910
		public string Tsd;

		// Token: 0x0400038F RID: 911
		private string CiteTsd;

		// Token: 0x04000390 RID: 912
		public string UpdatedTSD;

		// Token: 0x04000391 RID: 913
		public XElement FormDiffList;

		// Token: 0x04000392 RID: 914
		public XElement ActionDefaults;

		// Token: 0x04000393 RID: 915
		public string TiptopActionDefaults;

		// Token: 0x04000394 RID: 916
		public string CustomActionDefaults;

		// Token: 0x04000395 RID: 917
		public string ADT;

		// Token: 0x04000396 RID: 918
		public string infoXML;

		// Token: 0x04000397 RID: 919
		private XElement RefFormNode;

		// Token: 0x04000398 RID: 920
		public string ElementBindings;

		// Token: 0x02000111 RID: 273
		// (Invoke) Token: 0x060009D5 RID: 2517
		public delegate void TzpSavedHandler(object sender, TzpSavedEventArgs e);
	}
}
