using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Text;
using System.Text.RegularExpressions;
using System.Windows;
using System.Xml.Linq;
using SpecDesigner.Infrastructure.Helper;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Exceptions;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x02000024 RID: 36
	public class AddPointModel : IFunctionModel, INotifyPropertyChanged, IDisposable, IDataErrorInfo
	{
		// Token: 0x17000023 RID: 35
		// (get) Token: 0x06000080 RID: 128 RVA: 0x00002B1F File Offset: 0x00000D1F
		// (set) Token: 0x06000081 RID: 129 RVA: 0x00002B28 File Offset: 0x00000D28
		public string TglTag
		{
			get
			{
				return this._tglTag;
			}
			set
			{
				if (!string.IsNullOrEmpty(this._tglTag) && this._tglTag != value)
				{
					throw new Exception(string.Format(Application.Current.FindResource("Message_DuplicatedTag") as string, this.name));
				}
				this._tglTag = value;
			}
		}

		// Token: 0x17000024 RID: 36
		// (get) Token: 0x06000082 RID: 130 RVA: 0x00002B7C File Offset: 0x00000D7C
		public string SRC
		{
			get
			{
				return this._element.Attribute("src").Value;
			}
		}

		// Token: 0x06000083 RID: 131 RVA: 0x00002B98 File Offset: 0x00000D98
		public AddPointModel(PackageKey programKey, XElement source)
		{
			this.ID = Guid.NewGuid();
			this.ProgramKey = programKey;
			this._element = source;
		}

		// Token: 0x06000084 RID: 132 RVA: 0x00002C64 File Offset: 0x00000E64
		public AddPointModel(PackageKey programKey)
		{
			this.ProgramKey = programKey;
			this.Scope = Scope.PUBLIC;
			this.ID = Guid.NewGuid();
			this._status = Status.CREATE;
			this._element = new XElement("point");
			this._element.Add(new XCData(string.Empty));
			this._element.Add(new XAttribute("name", ""));
			this._element.Add(new XAttribute("order", ""));
			this._element.Add(new XAttribute("ver", ""));
			this._element.Add(new XAttribute("cite_std", YesNo.N));
			this._element.Add(new XAttribute("new", YesNo.Y));
			this._element.Add(new XAttribute("src", ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsTopstdMode ? "s" : ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ENV));
			this._element.Add(new XAttribute("status", Status.CREATE));
			this._element.Add(new XAttribute("ch", ""));
			this._element.Add(new XAttribute("ind_fun", ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Topind));
			this._element.Add(new XAttribute("ind_extra", "N"));
		}

		// Token: 0x17000025 RID: 37
		// (get) Token: 0x06000085 RID: 133 RVA: 0x00002ED8 File Offset: 0x000010D8
		// (set) Token: 0x06000086 RID: 134 RVA: 0x00002EE0 File Offset: 0x000010E0
		public Guid ID
		{
			get
			{
				return this.id;
			}
			private set
			{
				this.id = value;
			}
		}

		// Token: 0x17000026 RID: 38
		// (get) Token: 0x06000087 RID: 135 RVA: 0x00002EE9 File Offset: 0x000010E9
		// (set) Token: 0x06000088 RID: 136 RVA: 0x00002EF1 File Offset: 0x000010F1
		public PackageKey ProgramKey
		{
			get
			{
				return this.programKey;
			}
			set
			{
				this.programKey = value;
			}
		}

		// Token: 0x17000027 RID: 39
		// (get) Token: 0x06000089 RID: 137 RVA: 0x00002EFA File Offset: 0x000010FA
		public string FunctionNameWithoutParameter
		{
			get
			{
				return Regex.Replace(this.FunctionName, "\\([^\\)]*\\)", string.Empty, RegexOptions.IgnoreCase);
			}
		}

		// Token: 0x17000028 RID: 40
		// (get) Token: 0x0600008A RID: 138 RVA: 0x00002F12 File Offset: 0x00001112
		// (set) Token: 0x0600008B RID: 139 RVA: 0x00002F68 File Offset: 0x00001168
		public string FunctionName
		{
			get
			{
				return this.functionName;
			}
			set
			{
				if (this.Status == Status.DELETE)
				{
					this.functionName = value;
					return;
				}
				if (this.IsLoaded && (Status.DELETE & this.Status) == Status.NULL && this.functionName != value)
				{
					this.Status = Status.MODIFY;
					this.PublishModifiedEvent();
				}
				this.functionName = value;
				if (this._errors.ContainsKey("FunctionName"))
				{
					this._errors.Remove("FunctionName");
				}
				StringBuilder stringBuilder = new StringBuilder();
				if (this.FunctionName.Trim().Length == 0)
				{
					stringBuilder.Append(Application.Current.FindResource("Message_CantEmptyOrNull") as string);
				}
				else
				{
					if (this.FunctionName.Trim().IndexOf('(') == 0)
					{
						stringBuilder.Append(Application.Current.FindResource("Message_CantEmptyOrNull") as string);
					}
					if (!Regex.IsMatch(this.FunctionName, "(?:\\S+)(?:\\(+[^\\)]*\\)+)"))
					{
						if (stringBuilder.Length > 0)
						{
							stringBuilder.Append(Environment.NewLine);
						}
						stringBuilder.Append(Application.Current.FindResource("Message_FunctionNameFormat") as string);
					}
					if (Regex.IsMatch(this.functionName, "[^a-zA-Z0-9_(),]"))
					{
						if (stringBuilder.Length > 0)
						{
							stringBuilder.Append(Environment.NewLine);
						}
						stringBuilder.Append(Application.Current.FindResource("Message_FunctionNameFormat_1") as string);
					}
					IEnumerable<AddPointModel> enumerable = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffAddPoints.Where<AddPointModel>((AddPointModel a) => a.Name == this.Name && (a.Status & Status.DELETE) == Status.NULL && a.ID != this.ID);
					if (enumerable.Count<AddPointModel>() > 0)
					{
						if (stringBuilder.Length > 0)
						{
							stringBuilder.Append(Environment.NewLine);
						}
						stringBuilder.Append(Application.Current.FindResource("Message_NameMustUnique") as string);
					}
					else if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Nodes.Count > 0)
					{
						string cleanName = (this.functionName.Contains('(') ? this.functionName.Remove(this.functionName.IndexOf('(')) : this.functionName);
						IEnumerable<TreeItem> enumerable2 = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Nodes.First<TreeItem>().Nodes.Where<TreeItem>((TreeItem a) => a.Name == cleanName);
						if (enumerable2.Count<TreeItem>() > 0)
						{
							if (stringBuilder.Length > 0)
							{
								stringBuilder.Append(Environment.NewLine);
							}
							stringBuilder.Append(Application.Current.FindResource("Message_NameAlreadyExistInTemplate") as string);
						}
					}
					if (!this.FunctionName.StartsWith(this.ProgramKey.Program))
					{
						DocumentErrorsEventArgs e = new DocumentErrorsEventArgs();
						e.ProgramKey = this.ProgramKey;
						e.SourceType = this.ProgramKey.PackType;
						e.ErrorType = ErrorsType.INFORMATION;
						e.Time = DateTime.Now;
						e.Key = this.FunctionName;
						e.Description = string.Format("{0}：{1}", this.FunctionName, Application.Current.FindResource("Message_NameStartRule"));
						EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e);
					}
					if (this.FunctionName.Contains(" "))
					{
						if (stringBuilder.Length > 0)
						{
							stringBuilder.Append(Environment.NewLine);
						}
						stringBuilder.Append(Application.Current.FindResource("Message_NameNoBlackSpace") as string);
					}
					if (this.FunctionName.IndexOf('(') > AddPointModel.MAXNAMELENGTH)
					{
						if (stringBuilder.Length > 0)
						{
							stringBuilder.Append(Environment.NewLine);
						}
						stringBuilder.Append(Application.Current.FindResource("Message_NameMaxLengthRule") as string);
					}
				}
				if (stringBuilder.Length > 0)
				{
					this._errors.Add("FunctionName", stringBuilder.ToString());
				}
				this.OnPropertyChanged("FunctionName");
				this.SetName();
			}
		}

		// Token: 0x17000029 RID: 41
		// (get) Token: 0x0600008C RID: 140 RVA: 0x00003349 File Offset: 0x00001549
		// (set) Token: 0x0600008D RID: 141 RVA: 0x00003354 File Offset: 0x00001554
		public string Description
		{
			get
			{
				return this.description;
			}
			set
			{
				if (this._errors.ContainsKey("Description"))
				{
					this._errors.Remove("Description");
				}
				StringBuilder stringBuilder = new StringBuilder();
				if (!string.IsNullOrEmpty(this.description))
				{
					RegexOptions regexOptions = RegexOptions.IgnoreCase | RegexOptions.Multiline;
					Regex regex = new Regex("^\\s*#", regexOptions);
					string[] array = this.description.Split(new char[] { '\n' });
					for (int i = 0; i < array.Length; i++)
					{
						if (!regex.IsMatch(array[i]) && array[i].Trim() != string.Empty)
						{
							if (stringBuilder.Length > 0)
							{
								stringBuilder.Append(Environment.NewLine);
							}
							stringBuilder.Append(Application.Current.FindResource("Message_MustContainPoundBeforeLine") as string);
						}
					}
				}
				if (this.IsLoaded && (Status.DELETE & this.Status) == Status.NULL && this.description != value)
				{
					this.Status = Status.MODIFY;
					this.PublishModifiedEvent();
				}
				if (stringBuilder.Length > 0)
				{
					this._errors.Add("Description", stringBuilder.ToString());
				}
				this.description = value;
				this.OnPropertyChanged("Description");
			}
		}

		// Token: 0x1700002A RID: 42
		// (get) Token: 0x0600008E RID: 142 RVA: 0x00003485 File Offset: 0x00001685
		// (set) Token: 0x0600008F RID: 143 RVA: 0x00003494 File Offset: 0x00001694
		public string ContentData
		{
			get
			{
				return this.contentdata.ToString();
			}
			set
			{
				value = value.Replace("\r\n", "\n");
				string text = value;
				text = AddPointModel.scopeRegex.Replace(text, "", 1);
				if (AddPointModel.funcReg.IsMatch(text))
				{
					MatchCollection matchCollection = AddPointModel.funcReg.Matches(text);
					text = text.Remove(0, matchCollection[0].Groups[0].Index);
					text = AddPointModel.funcReg.Replace(text, "", 1);
					text = AddPointModel.endFuncReg.Replace(text, "", 1);
				}
				else if (AddPointModel.dialogReg.IsMatch(text))
				{
					MatchCollection matchCollection2 = AddPointModel.dialogReg.Matches(text);
					text = text.Remove(0, matchCollection2[0].Groups[0].Index);
					text = AddPointModel.dialogReg.Replace(text, "", 1);
					text = AddPointModel.enddialogReg.Replace(text, "", 1);
				}
				else if (AddPointModel.reportReg.IsMatch(text))
				{
					MatchCollection matchCollection3 = AddPointModel.reportReg.Matches(text);
					text = text.Remove(0, matchCollection3[0].Groups[0].Index);
					text = AddPointModel.reportReg.Replace(text, "", 1);
					text = AddPointModel.endreportReg.Replace(text, "", 1);
				}
				this.contentdata.Clear();
				this.contentdata.Append(text);
			}
		}

		// Token: 0x1700002B RID: 43
		// (get) Token: 0x06000090 RID: 144 RVA: 0x000035FB File Offset: 0x000017FB
		// (set) Token: 0x06000091 RID: 145 RVA: 0x00003608 File Offset: 0x00001808
		public string Content
		{
			get
			{
				return this.content.ToString();
			}
			set
			{
				value = value.Replace("\r\n", "\n");
				string text = value;
				if (this.IsSelfDefinition)
				{
					this.GetScope(ref text);
					if (string.IsNullOrEmpty(this.Name))
					{
						if (AddPointModel.funcReg.IsMatch(text))
						{
							this.Type = DefinitionType.FUNCTION;
						}
						else if (AddPointModel.dialogReg.IsMatch(text))
						{
							this.Type = DefinitionType.DIALOG;
						}
						else if (AddPointModel.reportReg.IsMatch(text))
						{
							this.Type = DefinitionType.REPORT;
						}
						else
						{
							this.Type = DefinitionType.NULL;
						}
					}
				}
				else
				{
					this.Type = DefinitionType.NULL;
				}
				switch (this.Type)
				{
				case DefinitionType.FUNCTION:
				{
					if (!AddPointModel.funcReg.IsMatch(text))
					{
						this.ThrowComplexException(value);
					}
					MatchCollection matchCollection = AddPointModel.funcReg.Matches(text);
					this.Description = text.Substring(0, matchCollection[0].Groups[0].Index).TrimEnd(new char[0]);
					text = text.Remove(0, matchCollection[0].Groups[0].Index);
					this.FunctionName = string.Format("{0}{1}", matchCollection[0].Groups["name"].Value, AddPointModel.noSpaceRegex.Replace(matchCollection[0].Groups["parameter"].Value, ""));
					text = AddPointModel.funcReg.Replace(text, "", 1);
					text = AddPointModel.endFuncReg.Replace(text, "", 1);
					break;
				}
				case DefinitionType.DIALOG:
				{
					if (!AddPointModel.dialogReg.IsMatch(text))
					{
						this.ThrowComplexException(value);
					}
					MatchCollection matchCollection2 = AddPointModel.dialogReg.Matches(text);
					this.Description = text.Substring(0, matchCollection2[0].Groups[0].Index).TrimEnd(new char[0]);
					text = text.Remove(0, matchCollection2[0].Groups[0].Index);
					this.FunctionName = string.Format("{0}{1}", matchCollection2[0].Groups["name"].Value, AddPointModel.noSpaceRegex.Replace(matchCollection2[0].Groups["parameter"].Value, ""));
					text = AddPointModel.dialogReg.Replace(text, "", 1);
					text = AddPointModel.enddialogReg.Replace(text, "", 1);
					break;
				}
				case DefinitionType.REPORT:
				{
					if (!AddPointModel.reportReg.IsMatch(text))
					{
						this.ThrowComplexException(value);
					}
					MatchCollection matchCollection3 = AddPointModel.reportReg.Matches(text);
					this.Description = text.Substring(0, matchCollection3[0].Groups[0].Index).TrimEnd(new char[0]);
					text = text.Remove(0, matchCollection3[0].Groups[0].Index);
					this.FunctionName = string.Format("{0}{1}", matchCollection3[0].Groups["name"].Value, AddPointModel.noSpaceRegex.Replace(matchCollection3[0].Groups["parameter"].Value, ""));
					text = AddPointModel.reportReg.Replace(text, "", 1);
					text = AddPointModel.endreportReg.Replace(text, "", 1);
					break;
				}
				}
				if (this.IsLoaded && (Status.DELETE & this.Status) == Status.NULL && !string.Equals(this.content.ToString().Replace("\n\a", "").Replace("\a\n", "")
					.Replace("\a", ""), text.ToString().Replace("\n\a", "").Replace("\a\n", "")
					.Replace("\a", "")))
				{
					this.Status |= Status.MODIFY;
					this.PublishModifiedEvent();
				}
				this.content.Clear();
				this.content.Append(text);
				this.OnPropertyChanged("Content");
			}
		}

		// Token: 0x06000092 RID: 146 RVA: 0x00003A3B File Offset: 0x00001C3B
		private void ThrowComplexException(string value)
		{
			throw new ComplexException(string.Format("函式內容錯誤：{0}", this.Name), string.Format("*************** Content ***************{0}{1}{0}*****************  End  *****************", Environment.NewLine, value));
		}

		// Token: 0x06000093 RID: 147 RVA: 0x00003A64 File Offset: 0x00001C64
		private void GetScope(ref string sb)
		{
			if (AddPointModel.scopeRegex.IsMatch(sb))
			{
				MatchCollection matchCollection = AddPointModel.scopeRegex.Matches(sb);
				string text;
				if ((text = matchCollection[0].Groups["type"].Value.ToUpper()) != null && text == "PRIVATE")
				{
					this.Scope = Scope.PRIVATE;
				}
				else
				{
					this.Scope = Scope.PUBLIC;
				}
				sb = AddPointModel.scopeRegex.Replace(sb, "", 1);
			}
		}

		// Token: 0x1700002C RID: 44
		// (get) Token: 0x06000094 RID: 148 RVA: 0x00003AE1 File Offset: 0x00001CE1
		// (set) Token: 0x06000095 RID: 149 RVA: 0x00003AE9 File Offset: 0x00001CE9
		public Scope Scope
		{
			get
			{
				return this._scope;
			}
			set
			{
				if (this.IsLoaded && this._scope != value)
				{
					this.Status |= Status.MODIFY;
					this.PublishModifiedEvent();
				}
				this._scope = value;
				this.OnPropertyChanged("Scope");
			}
		}

		// Token: 0x1700002D RID: 45
		// (get) Token: 0x06000096 RID: 150 RVA: 0x00003B22 File Offset: 0x00001D22
		// (set) Token: 0x06000097 RID: 151 RVA: 0x00003B2A File Offset: 0x00001D2A
		public bool IsLoaded
		{
			get
			{
				return this._isLoaded;
			}
			set
			{
				this._isLoaded = value;
			}
		}

		// Token: 0x1700002E RID: 46
		// (get) Token: 0x06000098 RID: 152 RVA: 0x00003B33 File Offset: 0x00001D33
		// (set) Token: 0x06000099 RID: 153 RVA: 0x00003B3B File Offset: 0x00001D3B
		public DefinitionType Type
		{
			get
			{
				return this.type;
			}
			set
			{
				if (this.type == value)
				{
					return;
				}
				this.type = value;
				this.SetName();
			}
		}

		// Token: 0x1700002F RID: 47
		// (get) Token: 0x0600009A RID: 154 RVA: 0x00003B54 File Offset: 0x00001D54
		// (set) Token: 0x0600009B RID: 155 RVA: 0x00003B5C File Offset: 0x00001D5C
		public IFunctionModel Parent
		{
			get
			{
				return this.parent;
			}
			set
			{
				this.parent = value;
				this.OnPropertyChanged("Parent");
			}
		}

		// Token: 0x17000030 RID: 48
		// (get) Token: 0x0600009C RID: 156 RVA: 0x00003B70 File Offset: 0x00001D70
		// (set) Token: 0x0600009D RID: 157 RVA: 0x00003B78 File Offset: 0x00001D78
		public YesNo IsMarkable
		{
			get
			{
				return this._isMarkable;
			}
			set
			{
				this._isMarkable = value;
			}
		}

		// Token: 0x17000031 RID: 49
		// (get) Token: 0x0600009E RID: 158 RVA: 0x00003B81 File Offset: 0x00001D81
		// (set) Token: 0x0600009F RID: 159 RVA: 0x00003B89 File Offset: 0x00001D89
		public string EditEnv
		{
			get
			{
				return this._editEnv;
			}
			set
			{
				this._editEnv = value;
			}
		}

		// Token: 0x17000032 RID: 50
		// (get) Token: 0x060000A0 RID: 160 RVA: 0x00003B92 File Offset: 0x00001D92
		// (set) Token: 0x060000A1 RID: 161 RVA: 0x00003B9C File Offset: 0x00001D9C
		public string Name
		{
			get
			{
				return this.name;
			}
			set
			{
				if (this.Status == Status.DELETE)
				{
					return;
				}
				if (this.name == value)
				{
					return;
				}
				this.name = value;
				this.OnPropertyChanged("Name");
				if (this.name.StartsWith("function.", StringComparison.InvariantCultureIgnoreCase))
				{
					this.Type = DefinitionType.FUNCTION;
					return;
				}
				if (this.name.StartsWith("dialog.", StringComparison.InvariantCultureIgnoreCase))
				{
					this.Type = DefinitionType.DIALOG;
					return;
				}
				if (this.name.StartsWith("report.", StringComparison.InvariantCultureIgnoreCase))
				{
					this.Type = DefinitionType.REPORT;
					return;
				}
				this.Type = DefinitionType.NULL;
			}
		}

		// Token: 0x17000033 RID: 51
		// (get) Token: 0x060000A2 RID: 162 RVA: 0x00003C2C File Offset: 0x00001E2C
		// (set) Token: 0x060000A3 RID: 163 RVA: 0x00003C34 File Offset: 0x00001E34
		public Status Status
		{
			get
			{
				return this._status;
			}
			set
			{
				this._status = value;
				this.ModifiedByTopstd = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsTopstdMode;
				this.SetAttribute("src", this.ModifiedByTopstd ? "s" : ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ENV);
				this.OnStatusChanged();
			}
		}

		// Token: 0x060000A4 RID: 164 RVA: 0x00003C9E File Offset: 0x00001E9E
		public void SetStatus(Status status)
		{
			this._status = status;
		}

		// Token: 0x17000034 RID: 52
		// (get) Token: 0x060000A5 RID: 165 RVA: 0x00003CA8 File Offset: 0x00001EA8
		public bool IsModified
		{
			get
			{
				string attribute = this.GetAttribute("ch");
				return (this.Status & Status.MODIFY) == Status.MODIFY || string.Equals("Y", attribute, StringComparison.CurrentCultureIgnoreCase);
			}
		}

		// Token: 0x17000035 RID: 53
		// (get) Token: 0x060000A6 RID: 166 RVA: 0x00003CDA File Offset: 0x00001EDA
		// (set) Token: 0x060000A7 RID: 167 RVA: 0x00003CE2 File Offset: 0x00001EE2
		public YesNo CiteSetting
		{
			get
			{
				return this._citeSetting;
			}
			set
			{
				if (value == this._citeSetting)
				{
					return;
				}
				this._citeSetting = value;
				if (this.IsLoaded && (Status.DELETE & this.Status) == Status.NULL)
				{
					this.Status = Status.MODIFY;
					this.PublishModifiedEvent();
				}
				this.OnPropertyChanged("CiteSetting");
			}
		}

		// Token: 0x17000036 RID: 54
		// (get) Token: 0x060000A8 RID: 168 RVA: 0x00003D1F File Offset: 0x00001F1F
		// (set) Token: 0x060000A9 RID: 169 RVA: 0x00003D27 File Offset: 0x00001F27
		public string Mapping
		{
			get
			{
				return this.mapping;
			}
			set
			{
				this.mapping = value;
				this.OnPropertyChanged("Mapping");
			}
		}

		// Token: 0x17000037 RID: 55
		// (get) Token: 0x060000AA RID: 170 RVA: 0x00003D3B File Offset: 0x00001F3B
		// (set) Token: 0x060000AB RID: 171 RVA: 0x00003D43 File Offset: 0x00001F43
		public string DiffBaseOnStandardModify
		{
			get
			{
				return this._diffBaseOnStandardModify;
			}
			set
			{
				this._diffBaseOnStandardModify = value;
			}
		}

		// Token: 0x17000038 RID: 56
		// (get) Token: 0x060000AC RID: 172 RVA: 0x00003D4C File Offset: 0x00001F4C
		// (set) Token: 0x060000AD RID: 173 RVA: 0x00003D54 File Offset: 0x00001F54
		public string Ind_fun
		{
			get
			{
				return this.ind_fun;
			}
			set
			{
				this.ind_fun = value;
				this.OnPropertyChanged("Ind");
			}
		}

		// Token: 0x17000039 RID: 57
		// (get) Token: 0x060000AE RID: 174 RVA: 0x00003D68 File Offset: 0x00001F68
		public bool IsEditable
		{
			get
			{
				if (SettingManager.Get().GetTzpManger(this.ProgramKey).isIndFun && this.Ind_fun != ResourceController.GetInstance().GetProgramInfo(this.programKey).Topind && this.Name != "global.memo_industry")
				{
					return false;
				}
				if (("sd" == ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Topind || string.IsNullOrEmpty(ResourceController.GetInstance().GetProgramInfo(this.programKey).Topind)) && "s" == ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ENV && this.Name == "global.memo_industry")
				{
					return false;
				}
				if (!ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsStandard && YesNo.Y == this.CiteSetting)
				{
					return false;
				}
				if (this._element.Attribute("readonly") != null && this._element.Attribute("readonly").Value.Equals("Y", StringComparison.CurrentCultureIgnoreCase))
				{
					return false;
				}
				string text;
				if ((text = this.EditEnv.ToLower()) != null)
				{
					if (!(text == "s"))
					{
						if (text == "c")
						{
							if ("s" == ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ENV)
							{
								return false;
							}
							if ("c" == ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ENV && ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsTopstdMode)
							{
								return false;
							}
						}
					}
					else if ("c" == ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ENV && !ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsTopstdMode && this.SRC.ToLower() != "c")
					{
						return false;
					}
				}
				string text2;
				if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsTopstdMode && (text2 = this.SRC.ToLower()) != null)
				{
					if (text2 == "c")
					{
						return this.Status == Status.CREATE;
					}
					if (text2 == "s" || text2 == "m")
					{
						return this.Status == Status.NULL || (this.ModifiedByTopstd && (this.Status & Status.MODIFY) == Status.MODIFY);
					}
				}
				return !("topstd" == ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).login_user) || !(this.SRC.ToLower() == "c") || this.Status == Status.CREATE;
			}
		}

		// Token: 0x1700003A RID: 58
		// (get) Token: 0x060000AF RID: 175 RVA: 0x0000403C File Offset: 0x0000223C
		public bool IsCustomized
		{
			get
			{
				return this.SRC == "c" || ((!this.ModifiedByTopstd || (this.Status & Status.MODIFY) != Status.MODIFY) && (!(ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).login_user == "topstd") || (this.Status & Status.MODIFY) != Status.MODIFY) && (this.Status & Status.MODIFY) != Status.NULL && (!(ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ENV == "s") || (this.Status & Status.MODIFY) != Status.MODIFY));
			}
		}

		// Token: 0x1700003B RID: 59
		// (get) Token: 0x060000B0 RID: 176 RVA: 0x000040DB File Offset: 0x000022DB
		// (set) Token: 0x060000B1 RID: 177 RVA: 0x000040F7 File Offset: 0x000022F7
		public bool ModifiedByTopstd
		{
			get
			{
				return "Y" == this.GetAttribute("modi_by_topstd");
			}
			set
			{
				this.SetAttribute("modi_by_topstd", value ? "Y" : "");
			}
		}

		// Token: 0x1700003C RID: 60
		// (get) Token: 0x060000B2 RID: 178 RVA: 0x00004114 File Offset: 0x00002314
		// (set) Token: 0x060000B3 RID: 179 RVA: 0x0000414C File Offset: 0x0000234C
		public int SortIndex
		{
			get
			{
				int num = int.MaxValue;
				if (!string.IsNullOrWhiteSpace(this.GetAttribute("order")))
				{
					num = int.Parse(this.GetAttribute("order"));
				}
				return num;
			}
			set
			{
				if (!this.IsSelfDefinition)
				{
					return;
				}
				if (this.GetAttribute("order").Equals(value.ToString(), StringComparison.InvariantCultureIgnoreCase))
				{
					return;
				}
				this.SetAttribute("order", value.ToString());
				this._status = this.Status | Status.MODIFY;
				string text;
				if ("c" == ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ENV && (text = this.SRC.ToLower()) != null && text == "s")
				{
					this.ModifiedByTopstd = true;
				}
				this.PublishModifiedEvent();
			}
		}

		// Token: 0x060000B4 RID: 180 RVA: 0x000041E8 File Offset: 0x000023E8
		private void SetName()
		{
			if (string.IsNullOrEmpty(this.FunctionName))
			{
				return;
			}
			switch (this.type)
			{
			case DefinitionType.FUNCTION:
				this.Name = string.Format("function.{0}", this.FunctionName.Split(new char[] { '(' })[0]);
				return;
			case DefinitionType.DIALOG:
				this.Name = string.Format("dialog.{0}", this.FunctionName.Split(new char[] { '(' })[0]);
				return;
			case DefinitionType.REPORT:
				this.Name = string.Format("report.{0}", this.FunctionName.Split(new char[] { '(' })[0]);
				return;
			default:
				throw new Exception("Unknown AddPoint Type");
			}
		}

		// Token: 0x1700003D RID: 61
		// (get) Token: 0x060000B5 RID: 181 RVA: 0x000042B1 File Offset: 0x000024B1
		public bool IsSelfDefinition
		{
			get
			{
				return this.Type != DefinitionType.NULL;
			}
		}

		// Token: 0x060000B6 RID: 182 RVA: 0x000042BF File Offset: 0x000024BF
		private void PublishModifiedEvent()
		{
			if (this.IsLoaded)
			{
				EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeChangedEvent>().Publish(this.ProgramKey);
			}
		}

		// Token: 0x060000B7 RID: 183 RVA: 0x000042E4 File Offset: 0x000024E4
		public void SetContent(string appendString)
		{
			if (this.content == null)
			{
				this.content = new StringBuilder();
			}
			this.content.Append(appendString);
		}

		// Token: 0x1700003E RID: 62
		// (get) Token: 0x060000B8 RID: 184 RVA: 0x00004306 File Offset: 0x00002506
		// (set) Token: 0x060000B9 RID: 185 RVA: 0x0000430E File Offset: 0x0000250E
		public string OriContent
		{
			get
			{
				return this._oriContent;
			}
			set
			{
				this._oriContent = value;
			}
		}

		// Token: 0x1700003F RID: 63
		// (get) Token: 0x060000BA RID: 186 RVA: 0x00004317 File Offset: 0x00002517
		// (set) Token: 0x060000BB RID: 187 RVA: 0x0000431F File Offset: 0x0000251F
		public int StartOffset
		{
			get
			{
				return this._startOffset;
			}
			set
			{
				this._startOffset = value;
			}
		}

		// Token: 0x17000040 RID: 64
		// (get) Token: 0x060000BC RID: 188 RVA: 0x00004328 File Offset: 0x00002528
		// (set) Token: 0x060000BD RID: 189 RVA: 0x00004330 File Offset: 0x00002530
		public int EndOffset
		{
			get
			{
				return this._endOffset;
			}
			set
			{
				this._endOffset = value;
			}
		}

		// Token: 0x060000BE RID: 190 RVA: 0x00004339 File Offset: 0x00002539
		public void Clear()
		{
			this.content.Clear();
		}

		// Token: 0x14000009 RID: 9
		// (add) Token: 0x060000BF RID: 191 RVA: 0x00004348 File Offset: 0x00002548
		// (remove) Token: 0x060000C0 RID: 192 RVA: 0x00004380 File Offset: 0x00002580
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060000C1 RID: 193 RVA: 0x000043B5 File Offset: 0x000025B5
		private void OnPropertyChanged(string name)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(name));
			}
		}

		// Token: 0x1400000A RID: 10
		// (add) Token: 0x060000C2 RID: 194 RVA: 0x000043D4 File Offset: 0x000025D4
		// (remove) Token: 0x060000C3 RID: 195 RVA: 0x0000440C File Offset: 0x0000260C
		public event PropertyChangedEventHandler StatusChanged;

		// Token: 0x060000C4 RID: 196 RVA: 0x00004441 File Offset: 0x00002641
		private void OnStatusChanged()
		{
			if (this.StatusChanged != null)
			{
				this.StatusChanged(this, new PropertyChangedEventArgs("Status"));
			}
		}

		// Token: 0x060000C5 RID: 197 RVA: 0x00004461 File Offset: 0x00002661
		private string GetAttribute(string key)
		{
			if (this._element.Attribute(key) != null)
			{
				return this._element.Attribute(key).Value;
			}
			return string.Empty;
		}

		// Token: 0x060000C6 RID: 198 RVA: 0x00004492 File Offset: 0x00002692
		private void SetAttribute(string key, string value)
		{
			this._element.SetAttributeValue(key, value);
			if (this.IsLoaded && string.Equals(key, "mark_hard", StringComparison.InvariantCultureIgnoreCase))
			{
				this.Status |= Status.MODIFY;
				this.PublishModifiedEvent();
			}
		}

		// Token: 0x17000041 RID: 65
		// (get) Token: 0x060000C7 RID: 199 RVA: 0x000044D0 File Offset: 0x000026D0
		// (set) Token: 0x060000C8 RID: 200 RVA: 0x000044E7 File Offset: 0x000026E7
		public bool IsMarkHard
		{
			get
			{
				return this.GetAttribute("mark_hard") == "Y";
			}
			set
			{
				this.SetAttribute("mark_hard", value ? "Y" : "N");
			}
		}

		// Token: 0x17000042 RID: 66
		// (get) Token: 0x060000C9 RID: 201 RVA: 0x00004503 File Offset: 0x00002703
		// (set) Token: 0x060000CA RID: 202 RVA: 0x0000451A File Offset: 0x0000271A
		public bool IsNew
		{
			get
			{
				return this.GetAttribute("new") == "Y";
			}
			set
			{
				this.SetAttribute("new", value ? "Y" : "N");
			}
		}

		// Token: 0x060000CB RID: 203 RVA: 0x00004538 File Offset: 0x00002738
		public AddPointModel Clone()
		{
			XElement xelement = this.ToXML();
			return new AddPointModel(this.ProgramKey, xelement)
			{
				ID = this.ID,
				Content = xelement.Value
			};
		}

		// Token: 0x060000CC RID: 204 RVA: 0x00004572 File Offset: 0x00002772
		public void Dispose()
		{
		}

		// Token: 0x060000CD RID: 205 RVA: 0x00004574 File Offset: 0x00002774
		public new string ToString()
		{
			StringBuilder stringBuilder = new StringBuilder();
			string text;
			if ((text = this.Type.ToString().ToUpper()) != null)
			{
				if (text == "FUNCTION")
				{
					stringBuilder.AppendLine(this.Description);
					stringBuilder.AppendLine(string.Format("{0} FUNCTION {1}", this.Scope, this.FunctionName));
					stringBuilder.AppendLine(this.Content);
					stringBuilder.Append("END FUNCTION");
					goto IL_01A4;
				}
				if (text == "DIALOG")
				{
					stringBuilder.AppendLine(this.Description);
					if (this.Scope == Scope.PUBLIC)
					{
						stringBuilder.AppendLine(string.Format("DIALOG {0}", this.FunctionName));
					}
					else
					{
						stringBuilder.AppendLine(string.Format("{0} DIALOG {1}", this.Scope, this.FunctionName));
					}
					stringBuilder.AppendLine(this.Content);
					stringBuilder.Append("END DIALOG");
					goto IL_01A4;
				}
				if (text == "REPORT")
				{
					stringBuilder.AppendLine(this.Description);
					if (this.Scope == Scope.PUBLIC)
					{
						stringBuilder.AppendLine(string.Format("REPORT {0}", this.FunctionName));
					}
					else
					{
						stringBuilder.AppendLine(string.Format("{0} REPORT {1}", this.Scope, this.FunctionName));
					}
					if (string.IsNullOrEmpty(this.Content))
					{
						this.content.Append("    FORMAT\r\n           \r\n        ON EVERY ROW\r\n            PRINTX g_grNumFmt.*\r\n            PRINTX");
					}
					stringBuilder.AppendLine(this.Content);
					stringBuilder.Append("END REPORT");
					goto IL_01A4;
				}
			}
			stringBuilder.Append(this.Content);
			IL_01A4:
			return stringBuilder.ToString();
		}

		// Token: 0x17000043 RID: 67
		public string this[string name]
		{
			get
			{
				if (this._errors.ContainsKey(name))
				{
					return this._errors[name];
				}
				return string.Empty;
			}
		}

		// Token: 0x17000044 RID: 68
		// (get) Token: 0x060000CF RID: 207 RVA: 0x0000474D File Offset: 0x0000294D
		public string Error
		{
			get
			{
				throw new NotImplementedException();
			}
		}

		// Token: 0x060000D0 RID: 208 RVA: 0x00004754 File Offset: 0x00002954
		public XElement ToXML()
		{
			XElement xelement = new XElement(this._element);
			char c = ' ';
			if (this.Status == Status.CREATE)
			{
				return null;
			}
			if ((this.Status & Status.DELETE) == Status.DELETE)
			{
				c = 'd';
			}
			else if ((this.Status & Status.MODIFY) == Status.MODIFY)
			{
				c = 'u';
			}
			xelement.SetAttributeValue("status", c.ToString());
			xelement.SetAttributeValue("cite_std", (this.CiteSetting == YesNo.Y) ? "Y" : "N");
			xelement.SetAttributeValue("name", this.Name);
			string text = (this.IsSelfDefinition ? this.ToString() : this.Content);
			if (SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff)
			{
				text = text.Replace("\n\a", "").Replace("\a", "");
			}
			xelement.ReplaceNodes(new XCData(text));
			return xelement;
		}

		// Token: 0x060000D1 RID: 209 RVA: 0x00004848 File Offset: 0x00002A48
		public static AddPointModel CreateDiffAddpoint(XElement xelement, PackageKey key)
		{
			AddPointModel addPointModel = new AddPointModel(key, xelement);
			string empty = string.Empty;
			foreach (XAttribute xattribute in xelement.Attributes())
			{
				string text;
				if ((text = xattribute.Name.LocalName.ToLower()) != null)
				{
					if (!(text == "name"))
					{
						if (text == "description")
						{
							addPointModel.Description = xattribute.Value;
						}
					}
					else
					{
						string value = xattribute.Value;
						addPointModel.Name = xattribute.Value;
					}
				}
				addPointModel.Content = xelement.Value;
			}
			return addPointModel;
		}

		// Token: 0x060000D2 RID: 210 RVA: 0x000048FC File Offset: 0x00002AFC
		public static AddPointModel Create(XElement xelement, PackageKey key)
		{
			AddPointModel addPointModel = new AddPointModel(key, xelement);
			string text = string.Empty;
			foreach (XAttribute xattribute in xelement.Attributes())
			{
				string text2;
				switch (text2 = xattribute.Name.LocalName.ToLower())
				{
				case "cite_std":
					if (xattribute.Value == "Y")
					{
						addPointModel.CiteSetting = YesNo.Y;
						continue;
					}
					continue;
				case "name":
					text = xattribute.Value;
					addPointModel.Name = xattribute.Value;
					continue;
				case "description":
					addPointModel.Description = xattribute.Value;
					continue;
				case "status":
					if (xattribute.Value.Equals(ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE), StringComparison.CurrentCultureIgnoreCase))
					{
						addPointModel.SetStatus(Status.DELETE);
						continue;
					}
					if (xattribute.Value.Equals(ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE), StringComparison.CurrentCultureIgnoreCase))
					{
						addPointModel.SetStatus(Status.CREATE);
						continue;
					}
					if (xattribute.Value.Equals(ReflectionHelpers.GetCustomDescription(SpecStatus.MODIFY), StringComparison.CurrentCultureIgnoreCase))
					{
						addPointModel.SetStatus(Status.MODIFY);
						continue;
					}
					addPointModel.SetStatus(Status.NULL);
					continue;
				case "mapping":
					addPointModel.Mapping = xattribute.Value;
					continue;
				case "ind_fun":
					addPointModel.Ind_fun = xattribute.Value;
					continue;
				}
				addPointModel.SetAttribute(xattribute.Name.LocalName, xattribute.Value);
			}
			if (addPointModel.Status == Status.DELETE)
			{
				addPointModel.SetContent(xelement.Value);
			}
			else
			{
				addPointModel.Content = xelement.Value;
				if (addPointModel.Type != DefinitionType.NULL && text != addPointModel.Name)
				{
					throw new ComplexException(string.Format(Application.Current.FindResource("Message_TapErrorCantRead") as string, text), xelement.Value);
				}
			}
			if (ResourceController.GetInstance().GetProgramInfo(key).IsDiff)
			{
				addPointModel.OriContent = addPointModel.Content;
			}
			return addPointModel;
		}

		// Token: 0x0400002C RID: 44
		private Dictionary<string, string> _errors = new Dictionary<string, string>();

		// Token: 0x0400002D RID: 45
		private static readonly int MAXNAMELENGTH = 48;

		// Token: 0x0400002E RID: 46
		private string _tglTag = string.Empty;

		// Token: 0x0400002F RID: 47
		private XElement _element;

		// Token: 0x04000030 RID: 48
		private Guid id;

		// Token: 0x04000031 RID: 49
		private PackageKey programKey;

		// Token: 0x04000032 RID: 50
		private static char[] LeagalChar = new char[] { '(', ')', ',', ' ', '_' };

		// Token: 0x04000033 RID: 51
		private string functionName = string.Empty;

		// Token: 0x04000034 RID: 52
		private string description = string.Empty;

		// Token: 0x04000035 RID: 53
		private static RegexOptions regexOptions = RegexOptions.IgnoreCase | RegexOptions.Multiline;

		// Token: 0x04000036 RID: 54
		private static Regex scopeRegex = new Regex("(?<type>^public|^private)", RegexOptions.IgnoreCase | RegexOptions.Multiline);

		// Token: 0x04000037 RID: 55
		private static Regex funcReg = new Regex("^\\s*(function)+\\s+(?<name>\\w+)*(?<parameter>\\([^\\)]*\\))(\\r{0,1})(\\n{0,1})", AddPointModel.regexOptions);

		// Token: 0x04000038 RID: 56
		private static Regex endFuncReg = new Regex("(\\r{0,1})(\\n{0,1})^(end function)\\s*$", AddPointModel.regexOptions);

		// Token: 0x04000039 RID: 57
		private static Regex dialogReg = new Regex("^\\s*(dialog)+\\s+(?<name>\\w+)*(?<parameter>\\([^\\)]*\\))(\\r{0,1})(\\n{0,1})", AddPointModel.regexOptions);

		// Token: 0x0400003A RID: 58
		private static Regex enddialogReg = new Regex("(\\r{0,1})(\\n{0,1})^(end dialog)\\s*$", AddPointModel.regexOptions);

		// Token: 0x0400003B RID: 59
		private static Regex reportReg = new Regex("^\\s*(report)+\\s+(?<name>\\w+)*(?<parameter>\\([^\\)]*\\))(\\r{0,1})(\\n{0,1})", AddPointModel.regexOptions);

		// Token: 0x0400003C RID: 60
		private static Regex endreportReg = new Regex("(\\r{0,1})(\\n{0,1})^(end report)\\s*$", AddPointModel.regexOptions);

		// Token: 0x0400003D RID: 61
		private static Regex noSpaceRegex = new Regex("\\s+", AddPointModel.regexOptions);

		// Token: 0x0400003E RID: 62
		private StringBuilder contentdata = new StringBuilder();

		// Token: 0x0400003F RID: 63
		private StringBuilder content = new StringBuilder();

		// Token: 0x04000040 RID: 64
		private Scope _scope = Scope.PUBLIC;

		// Token: 0x04000041 RID: 65
		private bool _isLoaded;

		// Token: 0x04000042 RID: 66
		private DefinitionType type = DefinitionType.FUNCTION;

		// Token: 0x04000043 RID: 67
		private IFunctionModel parent;

		// Token: 0x04000044 RID: 68
		private YesNo _isMarkable = YesNo.N;

		// Token: 0x04000045 RID: 69
		private string _editEnv = string.Empty;

		// Token: 0x04000046 RID: 70
		private string name = string.Empty;

		// Token: 0x04000047 RID: 71
		private Status _status;

		// Token: 0x04000048 RID: 72
		private YesNo _citeSetting = YesNo.N;

		// Token: 0x04000049 RID: 73
		private string mapping = string.Empty;

		// Token: 0x0400004A RID: 74
		private string _diffBaseOnStandardModify = "N";

		// Token: 0x0400004B RID: 75
		private string ind_fun = string.Empty;

		// Token: 0x0400004C RID: 76
		private string _oriContent = string.Empty;

		// Token: 0x0400004D RID: 77
		private int _startOffset;

		// Token: 0x0400004E RID: 78
		private int _endOffset;
	}
}
