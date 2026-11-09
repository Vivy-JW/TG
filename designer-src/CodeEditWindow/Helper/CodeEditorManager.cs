using System;
using System.Collections;
using System.Collections.Generic;
using System.Data;
using System.Linq;
using System.Text;
using System.Text.RegularExpressions;
using System.Windows;
using DifferenceEngine;
using ICSharpCode.AvalonEdit;
using ICSharpCode.AvalonEdit.Document;
using ICSharpCode.AvalonEdit.Editing;
using ICSharpCode.AvalonEdit.Highlighting;
using SpecDesigner.CodeEditWindow.View;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Event;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;
using SpecDesignerCommon.Connection;
using SpecDesignerCommon.Events;
using SpecDesignerPreference;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x0200003D RID: 61
	public class CodeEditorManager
	{
		// Token: 0x06000282 RID: 642 RVA: 0x000139FC File Offset: 0x00011BFC
		private CodeEditorManager(CodeEditorMainWindow view)
		{
			this.View = view;
		}

		// Token: 0x06000283 RID: 643 RVA: 0x00013A88 File Offset: 0x00011C88
		public CodeEditorManager(CodeEditorMainWindow view, PackageKey key)
			: this(view)
		{
			this.ProgramKey = key;
			EventController.GetInstance().GetEvent<LoadedSettingEvent>().Subscribe(new Action<LoadInformation>(this.Load));
		}

		// Token: 0x06000284 RID: 644 RVA: 0x00013AB4 File Offset: 0x00011CB4
		private void SubscribeEvent()
		{
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<FunctionSelectedEvent>().Subscribe(new Action<FunctionSelectedModel>(this.OnFunctionSelected));
			EventAggregatorManager.Global.GetEvent<SearchCodeEvent>().Subscribe(new Action<FieldArgs>(this.OnFieldSelected));
			EventAggregatorManager.Global.GetEvent<SaveSettingEvent>().Subscribe(new Action<PackageKey>(this.SaveFile));
			EventController.GetInstance().GetEvent<CreateFunctionEvent>().Subscribe(new Action<CreateFunctionContentArgs>(this.CreateFunction));
			EventController.GetInstance().GetEvent<DeleteFunctionEvent>().Subscribe(new Action<string>(this.DeleteFunction));
			EventController.GetInstance().GetEvent<ModifyFunctionEvent>().Subscribe(new Action<ModifyInfomation>(this.ModifyFunction));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.OnTzpFileClose));
			EventAggregatorManager.Global.GetEvent<ClosedSearchBoxEvent>().Subscribe(new Action<string>(this.OnClosedSearchBox));
			EventController.GetInstance().GetEvent<CreateFunctionModelEvent>().Subscribe(new Action<CreateFunctionInformation>(this.OnCreateFunctionModel));
			EventController.GetInstance().GetEvent<ModifyFunctionModelEvent>().Subscribe(new Action<ModifyFunctionInformation>(this.OnModifyFunctionModel));
			EventAggregatorManager.Global.GetEvent<RefreshScreen>().Subscribe(new Action<PackageKey>(this.OnRefreshScreen));
			EventAggregatorManager.Global.GetEvent<RefreshDiffResult>().Subscribe(new Action<PackageKey>(this.OnRefreshDiffResult));
			EventAggregatorManager.Global.GetEvent<FullTextRefreshScreen>().Subscribe(new Action<PackageKey>(this.OnFullTextRefreshScreen));
			EventAggregatorManager.Global.GetEvent<ToggleDiffEditableEvent>().Subscribe(new Action<PackageKey>(this.OnToggleDiffEditable));
			EventAggregatorManager.Global.GetEvent<SaveAddPointEvent>().Subscribe(new Action<PackageKey>(this.SaveADPContent));
		}

		// Token: 0x06000285 RID: 645 RVA: 0x00013C6C File Offset: 0x00011E6C
		private void UnsubscribeEvent()
		{
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<FunctionSelectedEvent>().Unsubscribe(new Action<FunctionSelectedModel>(this.OnFunctionSelected));
			EventAggregatorManager.Global.GetEvent<SearchCodeEvent>().Unsubscribe(new Action<FieldArgs>(this.OnFieldSelected));
			EventAggregatorManager.Global.GetEvent<SaveSettingEvent>().Unsubscribe(new Action<PackageKey>(this.SaveFile));
			EventController.GetInstance().GetEvent<CreateFunctionEvent>().Unsubscribe(new Action<CreateFunctionContentArgs>(this.CreateFunction));
			EventController.GetInstance().GetEvent<DeleteFunctionEvent>().Unsubscribe(new Action<string>(this.DeleteFunction));
			EventController.GetInstance().GetEvent<ModifyFunctionEvent>().Unsubscribe(new Action<ModifyInfomation>(this.ModifyFunction));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Unsubscribe(new Action<PackageKey>(this.OnTzpFileClose));
			EventAggregatorManager.Global.GetEvent<ClosedSearchBoxEvent>().Unsubscribe(new Action<string>(this.OnClosedSearchBox));
			EventController.GetInstance().GetEvent<CreateFunctionModelEvent>().Unsubscribe(new Action<CreateFunctionInformation>(this.OnCreateFunctionModel));
			EventController.GetInstance().GetEvent<ModifyFunctionModelEvent>().Unsubscribe(new Action<ModifyFunctionInformation>(this.OnModifyFunctionModel));
			EventAggregatorManager.Global.GetEvent<RefreshScreen>().Unsubscribe(new Action<PackageKey>(this.OnRefreshScreen));
			EventAggregatorManager.Global.GetEvent<RefreshDiffResult>().Unsubscribe(new Action<PackageKey>(this.OnRefreshDiffResult));
			EventAggregatorManager.Global.GetEvent<FullTextRefreshScreen>().Unsubscribe(new Action<PackageKey>(this.OnFullTextRefreshScreen));
			EventAggregatorManager.Global.GetEvent<ToggleDiffEditableEvent>().Unsubscribe(new Action<PackageKey>(this.OnToggleDiffEditable));
			EventAggregatorManager.Global.GetEvent<SaveAddPointEvent>().Unsubscribe(new Action<PackageKey>(this.SaveADPContent));
		}

		// Token: 0x1700005D RID: 93
		// (get) Token: 0x06000286 RID: 646 RVA: 0x00013E14 File Offset: 0x00012014
		// (set) Token: 0x06000287 RID: 647 RVA: 0x00013E1C File Offset: 0x0001201C
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x1700005E RID: 94
		// (get) Token: 0x06000288 RID: 648 RVA: 0x00013E35 File Offset: 0x00012035
		public RelayCommand ShowParametersCommand
		{
			get
			{
				if (this._showParametersCommand == null)
				{
					this._showParametersCommand = new RelayCommand(delegate(object param)
					{
						this.ShowParameters();
					}, (object param) => this.CanShow);
				}
				return this._showParametersCommand;
			}
		}

		// Token: 0x06000289 RID: 649 RVA: 0x00013E68 File Offset: 0x00012068
		private void ShowParameters()
		{
		}

		// Token: 0x1700005F RID: 95
		// (get) Token: 0x0600028A RID: 650 RVA: 0x00013E6A File Offset: 0x0001206A
		private bool CanShow
		{
			get
			{
				return false;
			}
		}

		// Token: 0x17000060 RID: 96
		// (get) Token: 0x0600028B RID: 651 RVA: 0x00013E6D File Offset: 0x0001206D
		// (set) Token: 0x0600028C RID: 652 RVA: 0x00013E75 File Offset: 0x00012075
		public CodeEditorMainWindow View
		{
			get
			{
				return this.view;
			}
			set
			{
				this.view = value;
				if (this.view != null)
				{
					this.Editor = this.view.textEditor;
				}
			}
		}

		// Token: 0x0600028D RID: 653 RVA: 0x00013E98 File Offset: 0x00012098
		private void main_VisualLinesChanged(object sender, EventArgs e)
		{
			if (!this.isScrollSynced)
			{
				return;
			}
			double y = this.Editor.TextArea.TextView.ScrollOffset.Y;
			double y2 = this._diff_editor.TextArea.TextView.ScrollOffset.Y;
			if (Math.Abs(y2 - y) > this._diff_editor.TextArea.TextView.DefaultLineHeight)
			{
				this._diff_editor.TextArea.ScrollToVerticalPosition(y);
			}
		}

		// Token: 0x0600028E RID: 654 RVA: 0x00013F1C File Offset: 0x0001211C
		private void diff_VisualLinesChanged(object sender, EventArgs e)
		{
			if (!this.isScrollSynced)
			{
				return;
			}
			double y = this.Editor.TextArea.TextView.ScrollOffset.Y;
			double y2 = this._diff_editor.TextArea.TextView.ScrollOffset.Y;
			if (Math.Abs(y2 - y) > this.Editor.TextArea.TextView.DefaultLineHeight)
			{
				this.Editor.TextArea.ScrollToVerticalPosition(y2);
			}
		}

		// Token: 0x0600028F RID: 655 RVA: 0x00013FA0 File Offset: 0x000121A0
		private void main_ScrollOffsetChanged(object sender, EventArgs e)
		{
			if (!this.isScrollSynced)
			{
				return;
			}
			if (!this.isScrollSynced)
			{
				return;
			}
			double y = this.Editor.TextArea.TextView.ScrollOffset.Y;
			double y2 = this._diff_editor.TextArea.TextView.ScrollOffset.Y;
			if (Math.Abs(y2 - y) > this._diff_editor.TextArea.TextView.DefaultLineHeight)
			{
				this._diff_editor.TextArea.ScrollToVerticalPosition(y);
			}
			double x = this.Editor.TextArea.TextView.ScrollOffset.X;
			double x2 = this._diff_editor.TextArea.TextView.ScrollOffset.X;
			this._diff_editor.TextArea.ScrollToHorizontalPosition(x);
		}

		// Token: 0x06000290 RID: 656 RVA: 0x0001407C File Offset: 0x0001227C
		private void diff_ScrollOffsetChanged(object sender, EventArgs e)
		{
			if (!this.isScrollSynced)
			{
				return;
			}
			double y = this.Editor.TextArea.TextView.ScrollOffset.Y;
			double y2 = this._diff_editor.TextArea.TextView.ScrollOffset.Y;
			if (Math.Abs(y2 - y) > this.Editor.TextArea.TextView.DefaultLineHeight)
			{
				this.Editor.TextArea.ScrollToVerticalPosition(y2);
			}
			double x = this.Editor.TextArea.TextView.ScrollOffset.X;
			double x2 = this._diff_editor.TextArea.TextView.ScrollOffset.X;
			this.Editor.TextArea.ScrollToHorizontalPosition(x2);
		}

		// Token: 0x17000061 RID: 97
		// (get) Token: 0x06000291 RID: 657 RVA: 0x0001414E File Offset: 0x0001234E
		// (set) Token: 0x06000292 RID: 658 RVA: 0x0001417C File Offset: 0x0001237C
		public CodeTextEditor Editor
		{
			get
			{
				if (this._editor == null && this.View.textEditor != null)
				{
					this._editor = this.View.textEditor;
				}
				return this._editor;
			}
			set
			{
				if (this._editor != null)
				{
					this._editor.TextChanged -= this.Editor_TextChanged;
				}
				this._editor = value;
				if (this._editor != null)
				{
					this._editor.TextChanged += this.Editor_TextChanged;
				}
			}
		}

		// Token: 0x06000293 RID: 659 RVA: 0x000141D0 File Offset: 0x000123D0
		private void Editor_TextChanged(object sender, EventArgs e)
		{
			if (!this.Editor.IsContentLoaded)
			{
				return;
			}
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeChangedEvent>().Publish(this.ProgramKey);
			ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).FullContext = this.Editor.Text;
		}

		// Token: 0x17000062 RID: 98
		// (get) Token: 0x06000294 RID: 660 RVA: 0x00014226 File Offset: 0x00012426
		// (set) Token: 0x06000295 RID: 661 RVA: 0x0001422E File Offset: 0x0001242E
		public DiffTextViewer Diff_Editor
		{
			get
			{
				return this._diff_editor;
			}
			set
			{
				this._diff_editor = value;
			}
		}

		// Token: 0x06000296 RID: 662 RVA: 0x00014237 File Offset: 0x00012437
		public IEnumerable<Guid> GetDiffSegments()
		{
			return this._DiffSectionList.Distinct<Guid>();
		}

		// Token: 0x06000297 RID: 663 RVA: 0x00014244 File Offset: 0x00012444
		private void SaveFile(PackageKey key)
		{
			if (this.ProgramKey != key)
			{
				return;
			}
			this.SaveContent();
			this.GenerateTGL();
			ProgramInformation programInfo = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey);
			programInfo.FullCode = this.Editor.Text;
			if (programInfo.IsDiff)
			{
				if (this.Diff_Editor.IsChangedMode)
				{
					bool flag;
					if (PreferenceManager.Current.Settings.RemaindDiffSave)
					{
						flag = MessageBoxResult.Yes == DesignerMessageBox.Show(Application.Current.FindResource("CE_ConfirmDiffSave") as string, Application.Current.FindResource("Message_Message") as string, MessageBoxButton.YesNo);
					}
					else
					{
						flag = PreferenceManager.Current.Settings.AlwaysSaveDiff;
					}
					if (flag)
					{
						SettingManager.Get().GetTzpManger(this.ProgramKey).SaveDiffSource(this.Diff_Editor.Text.Replace("\a\r\n", "").Replace("\a", ""));
						this.Diff_Editor.IsChangedMode = false;
						return;
					}
				}
				SettingManager.Get().GetTzpManger(this.ProgramKey).SaveDiffSource();
			}
		}

		// Token: 0x06000298 RID: 664 RVA: 0x00014367 File Offset: 0x00012567
		internal void SaveContent()
		{
			this.SaveADPContent();
			if (this.Editor.Mode == ContentType.SEC)
			{
				this.SaveSectionContent();
			}
		}

		// Token: 0x06000299 RID: 665 RVA: 0x000143DC File Offset: 0x000125DC
		private void GenerateTGL()
		{
			if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Sections.Where<SectionModel>((SectionModel sec) => sec.Status.Equals("u", StringComparison.CurrentCultureIgnoreCase)).Count<SectionModel>() == 0)
			{
				return;
			}
			string tgl = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).TGL;
			StringBuilder stringBuilder = new StringBuilder(tgl);
			MatchCollection matchCollection = Regex.Matches(tgl, this.SectionStartPattern, RegexOptions.Multiline);
			MatchCollection matchCollection2 = Regex.Matches(tgl, this.SectionEndPattern, RegexOptions.Multiline);
			if (matchCollection.Count != matchCollection2.Count)
			{
				throw new Exception(Application.Current.FindResource("Message_SessionErrorCantSave") as string);
			}
			for (int i = 0; i < matchCollection.Count; i++)
			{
				Match startMatch = matchCollection[i];
				Match match = matchCollection2[i];
				int index = startMatch.Index;
				int index2 = match.Index;
				int length = match.Length;
				string text = stringBuilder.ToString(index, match.Index + match.Length - startMatch.Index);
				SectionModel sectionModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Sections.Where<SectionModel>((SectionModel sec) => sec.Name == startMatch.Groups["name"].Value && sec.Status.Equals("u", StringComparison.CurrentCultureIgnoreCase)).ElementAtOrDefault<SectionModel>(0);
				if (sectionModel != null)
				{
					string text2 = string.Format("{0}{1}{2}{1}{3}", new object[]
					{
						startMatch.Value,
						Environment.NewLine,
						sectionModel.Content,
						match.Value
					});
					stringBuilder.Replace(text, text2);
					matchCollection = Regex.Matches(stringBuilder.ToString(), this.SectionStartPattern, RegexOptions.Multiline);
					matchCollection2 = Regex.Matches(stringBuilder.ToString(), this.SectionEndPattern, RegexOptions.Multiline);
				}
			}
			ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).TGL = stringBuilder.ToString();
		}

		// Token: 0x0600029A RID: 666 RVA: 0x000145D2 File Offset: 0x000127D2
		public void SaveADPContent(PackageKey key)
		{
			this.SaveADPContent();
		}

		// Token: 0x0600029B RID: 667 RVA: 0x000145F0 File Offset: 0x000127F0
		internal void SaveADPContent()
		{
			IEnumerable<AddPointModel> enumerable = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => ((a.Status & Status.DELETE) == Status.NULL) & a.IsLoaded);
			for (int i = 0; i < enumerable.Count<AddPointModel>(); i++)
			{
				AddPointModel addPointModel = enumerable.ElementAt<AddPointModel>(i);
				try
				{
					if (SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff)
					{
						if (this.Editor.TextArea.Document.SectionProvider.GetText(addPointModel.ID).Contains("\a"))
						{
							addPointModel.Content = this.Editor.TextArea.Document.SectionProvider.GetText(addPointModel.ID).Replace("\a\r\n", "").Replace("\a", "");
						}
						else
						{
							addPointModel.Content = this.Editor.TextArea.Document.SectionProvider.GetText(addPointModel.ID);
						}
					}
					else
					{
						addPointModel.Content = this.Editor.TextArea.Document.SectionProvider.GetText(addPointModel.ID);
					}
				}
				catch
				{
					DocumentErrorsEventArgs e = new DocumentErrorsEventArgs();
					e.ProgramKey = this.ProgramKey;
					e.SourceType = this.ProgramKey.PackType;
					e.ErrorType = ErrorsType.ERROR;
					e.Time = DateTime.Now;
					e.Key = this.ProgramKey.Program;
					e.Description = string.Format(Application.Current.FindResource("Message_SaveError") as string, addPointModel.Name);
					EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e);
				}
			}
		}

		// Token: 0x0600029C RID: 668 RVA: 0x000147FC File Offset: 0x000129FC
		public void SaveSectionContent()
		{
			IEnumerable<Guid> dirtySegments = this.Editor.Document.SectionProvider.GetDirtySegments();
			List<int> list = new List<int>();
			if (dirtySegments.Count<Guid>() > 0)
			{
				if (dirtySegments.Count<Guid>() > 0)
				{
					ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsSectionModify = true;
				}
				Guid id;
				foreach (Guid guid in dirtySegments)
				{
					id = guid;
					SectionModel sectionModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Sections.Where<SectionModel>((SectionModel sec) => sec.ID == id).ElementAtOrDefault<SectionModel>(0);
					try
					{
						if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Type == "G" && ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsSectionModify)
						{
							string text = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ProgramKey.Program + ".other_function";
							string text2 = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ProgramKey.Program + ".other_report";
							if ((sectionModel.Name == text || sectionModel.Name == text2) && sectionModel.IsReadOnly)
							{
								continue;
							}
						}
						else if (sectionModel.IsReadOnly)
						{
							continue;
						}
						SegmentObject segmentObject = this.Editor.Document.SectionProvider.Find(ContentType.SEC, id);
						int i = segmentObject.Offset;
						bool flag = false;
						StringBuilder stringBuilder = new StringBuilder();
						while (i <= segmentObject.EndOffset)
						{
							string text3 = string.Empty;
							SegmentObject adp = this.Editor.Document.SectionProvider.Find(i);
							if (adp != null && adp is EditObject)
							{
								if (!flag)
								{
									AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => ap.ID == adp.ID).ElementAtOrDefault<AddPointModel>(0);
									flag = true;
									text3 = addPointModel.TglTag;
									if (addPointModel.IsMarkHard)
									{
										addPointModel.IsMarkHard = false;
										list.Add(this.Editor.Document.GetLineByNumber(i).Offset);
									}
									if (stringBuilder.Length > 0)
									{
										stringBuilder.AppendLine();
									}
									stringBuilder.Append(text3);
								}
							}
							else
							{
								flag = false;
								int offset = this.Editor.Document.GetLineByNumber(i).Offset;
								int endOffset = this.Editor.Document.GetLineByNumber(i).EndOffset;
								text3 = this.Editor.Document.GetText(offset, endOffset - offset);
								if (stringBuilder.Length > 0)
								{
									stringBuilder.AppendLine();
								}
								stringBuilder.Append(text3);
							}
							i++;
						}
						sectionModel.Content = stringBuilder.ToString().Replace("\a\r\n", "").Replace("\r\n\a", "");
					}
					catch
					{
						DocumentErrorsEventArgs e = new DocumentErrorsEventArgs();
						e.ProgramKey = this.ProgramKey;
						e.SourceType = this.ProgramKey.PackType;
						e.ErrorType = ErrorsType.ERROR;
						e.Time = DateTime.Now;
						e.Key = this.ProgramKey.Program;
						e.Description = string.Format(Application.Current.FindResource("Message_SaveError") as string, sectionModel.Name);
						EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e);
					}
				}
				this.Editor.Document.SectionProvider.ClearDirtySegments();
				foreach (int num in list)
				{
					this.Editor.RemoveMarkedArea(num);
				}
				EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeChangedEvent>().Publish(this.ProgramKey);
				if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ENV == "c" && SettingManager.Get().GetTzpManger(this.ProgramKey).Booking)
				{
					ConnectionManager.RunProgram(string.Format("{0} {1}", "adzi520", this.ProgramKey.Program));
				}
			}
		}

		// Token: 0x17000063 RID: 99
		// (get) Token: 0x0600029D RID: 669 RVA: 0x00014CEC File Offset: 0x00012EEC
		public List<DiffColor> DiffColorAreaList
		{
			get
			{
				return new List<DiffColor>(this._diffColorAreaList);
			}
		}

		// Token: 0x0600029E RID: 670 RVA: 0x00014D14 File Offset: 0x00012F14
		public void Diff()
		{
			this._diffColorAreaList.Clear();
			this.Diff_Editor.ClearRenderRegion();
			this.Editor.ClearRenderRegion();
			DiffList_TextFile diffList_TextFile = new DiffList_TextFile(this.Diff_Editor.Text.Replace("\a\r\n", "").Replace("\a", ""));
			DiffList_TextFile diffList_TextFile2 = new DiffList_TextFile(this.Editor.Text.Replace("\a\r\n", "").Replace("\a", ""));
			DiffEngine diffEngine = new DiffEngine();
			diffEngine.ProcessDiff(diffList_TextFile, diffList_TextFile2, DiffEngineLevel.SlowPerfect);
			ArrayList arrayList = diffEngine.DiffReport();
			StringBuilder stringBuilder = new StringBuilder();
			int num = 0;
			int num2 = 0;
			int num3 = 0;
			int num4 = 0;
			foreach (object obj in arrayList)
			{
				DiffResultSpan diffResultSpan = (DiffResultSpan)obj;
				switch (diffResultSpan.Status)
				{
				case DiffResultSpanStatus.NoChange:
				{
					for (int i = 0; i < diffResultSpan.Length; i++)
					{
						if (num < diffList_TextFile.Count() - 1)
						{
							stringBuilder.Append(diffList_TextFile.OriSource[num].ToString().Replace("\r", "") + "\r\n");
						}
						else
						{
							stringBuilder.Append(diffList_TextFile.OriSource[num].ToString().Replace("\r", ""));
						}
						num++;
						num2++;
					}
					break;
				}
				case DiffResultSpanStatus.Replace:
				{
					bool flag = false;
					int j = 0;
					while (j < diffResultSpan.Length)
					{
						if (num < diffList_TextFile.Count() - 1 || num2 < diffList_TextFile2.Count() - 1)
						{
							stringBuilder.Append(diffList_TextFile.OriSource[num].ToString().Replace("\r", "") + "\r\n");
						}
						else
						{
							stringBuilder.Append(diffList_TextFile.OriSource[num].ToString().Replace("\r", ""));
						}
						bool flag2 = false;
						SegmentObject segmentObject = this.Editor.Document.SectionProvider.Find(ContentType.ADP, num2 + num4 + 1);
						if (segmentObject is DummyObject)
						{
							segmentObject = this.Editor.Document.SectionProvider.Find(ContentType.SEC, num2 + num4 + 1);
							if (segmentObject is SectionObject)
							{
								flag2 = true;
							}
						}
						if ((flag2 && PreferenceManager.Current.Settings.DiffWithoutSection) || num + 1 >= diffList_TextFile.Count() || num2 + 1 >= diffList_TextFile2.Count())
						{
							goto IL_0A73;
						}
						if (diffList_TextFile.OriSource[num].ToString().Replace("\r", "").ToLower()
							.Equals(diffList_TextFile2.GetByIndex(num2).ToString().Replace("\r", "")
								.ToLower()))
						{
							num++;
							num2++;
						}
						else if (PreferenceManager.Current.Settings.DiffWithANSI && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "UNIQUE", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "DISTINCT", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "sysdate", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "CURRENT_DATE", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "TRUNC(SYSTIMESTAMP)", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "SYSTIMESTAMP", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "CURRENT_TIMESTAMP", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "UNIQUE", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "DISTINCT", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "sysdate", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "CURRENT_DATE", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "TRUNC(SYSTIMESTAMP)", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "SYSTIMESTAMP", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "CURRENT_TIMESTAMP", RegexOptions.IgnoreCase))
						{
							num++;
							num2++;
						}
						else
						{
							if (Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*{<") || Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "^\\s*{<") || Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*#此ctrlp無內容"))
							{
								goto IL_0A73;
							}
							if ((Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*#") && Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "^\\s*#")) || (Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*#") && string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""))) || (string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "^\\s*#")))
							{
								if (!PreferenceManager.Current.Settings.DiffComment)
								{
									goto IL_0A73;
								}
								if (!string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && !string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
								{
									if (!PreferenceManager.Current.Settings.SimplifyColor)
									{
										this.Diff_Editor.AddRenderRegion(num + 1 + num3, DiffResultSpanStatus.Replace);
										this.Editor.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.Replace);
									}
									flag = true;
									goto IL_0A73;
								}
								if (!string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
								{
									if (!PreferenceManager.Current.Settings.SimplifyColor)
									{
										this.Editor.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.DeleteSource);
									}
									flag = true;
									goto IL_0A73;
								}
								if (string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && !string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
								{
									if (!PreferenceManager.Current.Settings.SimplifyColor)
									{
										this.Diff_Editor.AddRenderRegion(num + 1 + num3, DiffResultSpanStatus.DeleteSource);
									}
									flag = true;
									goto IL_0A73;
								}
								goto IL_0A73;
							}
							else
							{
								if (!string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && !string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
								{
									if (!PreferenceManager.Current.Settings.SimplifyColor)
									{
										this.Diff_Editor.AddRenderRegion(num + 1 + num3, DiffResultSpanStatus.Replace);
										this.Editor.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.Replace);
									}
									flag = true;
									goto IL_0A73;
								}
								if (!string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
								{
									if (!PreferenceManager.Current.Settings.SimplifyColor)
									{
										this.Editor.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.DeleteSource);
									}
									flag = true;
									goto IL_0A73;
								}
								if (string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")) && !string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
								{
									if (!PreferenceManager.Current.Settings.SimplifyColor)
									{
										this.Diff_Editor.AddRenderRegion(num + 1 + num3, DiffResultSpanStatus.DeleteSource);
									}
									flag = true;
									goto IL_0A73;
								}
								goto IL_0A73;
							}
						}
						IL_0AA5:
						j++;
						continue;
						IL_0A73:
						if (flag)
						{
							DiffColor diffColor = new DiffColor();
							diffColor.LineNumber = num + 1 + num3;
							this._diffColorAreaList.Add(diffColor);
						}
						num++;
						num2++;
						goto IL_0AA5;
					}
					break;
				}
				case DiffResultSpanStatus.DeleteSource:
				{
					bool flag = false;
					bool flag3 = false;
					StringBuilder stringBuilder2 = new StringBuilder();
					int num5 = 0;
					bool flag4 = false;
					bool flag5 = false;
					int num6 = 1;
					SegmentObject segmentObject2 = this.Editor.Document.SectionProvider.Find(ContentType.ADP, num2 + 1 + num4);
					if (segmentObject2 is DummyObject || null == segmentObject2)
					{
						SegmentObject segmentObject3 = this.Editor.Document.SectionProvider.Find(ContentType.SEC, num2 + 1 + num4);
						if (segmentObject3 != null)
						{
							SegmentObject segmentObject4 = this.Editor.Document.SectionProvider.Find(ContentType.ADP, num2 + 1 + num4 - 1);
							if (null != segmentObject4 && segmentObject4 is EditObject)
							{
								this.Editor.Mode = ContentType.ADP | ContentType.OPEN;
								flag4 = true;
							}
							else
							{
								this.Editor.Mode = ContentType.SEC | ContentType.OPEN;
							}
						}
						else
						{
							SegmentObject segmentObject5 = this.Editor.Document.SectionProvider.Find(ContentType.SEC, num2 + 2 + num4);
							if (segmentObject5 == null)
							{
								num6 = 1;
								while (segmentObject3 == null)
								{
									segmentObject3 = this.Editor.Document.SectionProvider.Find(ContentType.SEC, num2 - num6 + num4);
									if (segmentObject3 == null)
									{
										num6++;
									}
								}
								this.Editor.Mode = ContentType.SEC | ContentType.OPEN;
								flag5 = true;
							}
						}
					}
					else
					{
						this.Editor.Mode = ContentType.ADP | ContentType.OPEN;
					}
					int k = 0;
					while (k < diffResultSpan.Length)
					{
						if (num < diffList_TextFile.Count() - 1 || num2 < diffList_TextFile2.Count() - 1)
						{
							stringBuilder.Append(diffList_TextFile.OriSource[num].ToString().Replace("\r", "") + "\r\n");
						}
						else
						{
							stringBuilder.Append(diffList_TextFile.OriSource[num].ToString().Replace("\r", ""));
						}
						if (flag4 && this.Editor.Mode == (ContentType.ADP | ContentType.OPEN))
						{
							if (!PreferenceManager.Current.Settings.DiffWithANSI || Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "UNIQUE", RegexOptions.IgnoreCase) || Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "DISTINCT", RegexOptions.IgnoreCase) || Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "sysdate", RegexOptions.IgnoreCase) || Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "CURRENT_DATE", RegexOptions.IgnoreCase) || Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "TRUNC(SYSTIMESTAMP)", RegexOptions.IgnoreCase) || Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "SYSTIMESTAMP", RegexOptions.IgnoreCase) || Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "CURRENT_TIMESTAMP", RegexOptions.IgnoreCase))
							{
								if (!Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*{<") && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*#此ctrlp無內容") && !string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")))
								{
									if (Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*#"))
									{
										if (!PreferenceManager.Current.Settings.SimplifyColor)
										{
											this.Editor.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.DeleteSource);
										}
										flag = true;
									}
									else
									{
										if (!PreferenceManager.Current.Settings.SimplifyColor)
										{
											this.Editor.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.DeleteSource);
										}
										flag = true;
									}
								}
								if (!flag3)
								{
									num5 = this.Editor.Document.GetLineByNumber(num2 + 1 + num4 - 1).EndOffset;
									flag3 = true;
								}
								stringBuilder2.Append("\r\n\a");
								goto IL_18D2;
							}
							num4++;
							num++;
						}
						else
						{
							if (flag5 && this.Editor.Mode == (ContentType.SEC | ContentType.OPEN))
							{
								if (!PreferenceManager.Current.Settings.DiffWithoutSection)
								{
									if (PreferenceManager.Current.Settings.DiffWithANSI && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "UNIQUE", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "DISTINCT", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "sysdate", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "CURRENT_DATE", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "TRUNC(SYSTIMESTAMP)", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "SYSTIMESTAMP", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "CURRENT_TIMESTAMP", RegexOptions.IgnoreCase))
									{
										num4++;
										num++;
										goto IL_1904;
									}
									if (!Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*{<") && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*#此ctrlp無內容") && !string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")))
									{
										if (Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*#"))
										{
											if (PreferenceManager.Current.Settings.DiffComment)
											{
												if (!PreferenceManager.Current.Settings.SimplifyColor)
												{
													this.Editor.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.DeleteSource);
												}
												flag = true;
											}
										}
										else
										{
											if (!PreferenceManager.Current.Settings.SimplifyColor)
											{
												this.Editor.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.DeleteSource);
											}
											flag = true;
										}
									}
								}
								if (!flag3)
								{
									num5 = this.Editor.Document.GetOffset(num2 - num6 + num4, 0);
									flag3 = true;
								}
								stringBuilder2.Append(string.Format("{0}{1}", "\a", Environment.NewLine));
								goto IL_18D2;
							}
							bool flag6 = false;
							SegmentObject segmentObject6 = this.Editor.Document.SectionProvider.Find(ContentType.ADP, num2 + num4 + 1);
							if (segmentObject6 is DummyObject)
							{
								segmentObject6 = this.Editor.Document.SectionProvider.Find(ContentType.SEC, num2 + num4 + 1);
								if (segmentObject6 is SectionObject)
								{
									flag6 = true;
								}
							}
							if (!flag6 || !PreferenceManager.Current.Settings.DiffWithoutSection)
							{
								if (PreferenceManager.Current.Settings.DiffWithANSI && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "UNIQUE", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "DISTINCT", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "sysdate", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "CURRENT_DATE", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "TRUNC(SYSTIMESTAMP)", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "SYSTIMESTAMP", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "CURRENT_TIMESTAMP", RegexOptions.IgnoreCase))
								{
									num4++;
									num++;
									goto IL_1904;
								}
								if (!Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*{<") && !Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*#此ctrlp無內容") && !string.IsNullOrEmpty(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", "")))
								{
									if (Regex.IsMatch(((TextLine)diffList_TextFile.GetByIndex(num)).Line.Replace("\a", ""), "^\\s*#"))
									{
										if (!PreferenceManager.Current.Settings.SimplifyColor)
										{
											this.Editor.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.DeleteSource);
										}
										flag = true;
									}
									else
									{
										if (!PreferenceManager.Current.Settings.SimplifyColor)
										{
											this.Editor.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.DeleteSource);
										}
										flag = true;
									}
								}
							}
							if (!flag3)
							{
								num5 = this.Editor.Document.GetOffset(num2 + 1 + num4, 0);
								flag3 = true;
							}
							stringBuilder2.Append(string.Format("{0}{1}", "\a", Environment.NewLine));
							goto IL_18D2;
						}
						IL_1904:
						k++;
						continue;
						IL_18D2:
						if (flag)
						{
							DiffColor diffColor2 = new DiffColor();
							diffColor2.LineNumber = num + 1 + num3;
							this._diffColorAreaList.Add(diffColor2);
						}
						num4++;
						num++;
						goto IL_1904;
					}
					this.Editor.Document.Insert(num5, stringBuilder2.ToString());
					if (this.Editor.Mode == (ContentType.SEC | ContentType.OPEN))
					{
						this.Editor.Mode = ContentType.SEC;
					}
					else if (this.Editor.Mode == (ContentType.ADP | ContentType.OPEN))
					{
						this.Editor.Mode = ContentType.ADP;
					}
					break;
				}
				case DiffResultSpanStatus.AddDestination:
				{
					bool flag = false;
					int l = 0;
					while (l < diffResultSpan.Length)
					{
						if (num < diffList_TextFile.Count() - 1 || num2 < diffList_TextFile2.Count() - 1)
						{
							stringBuilder.Append("\a\r\n");
						}
						else
						{
							stringBuilder.Append("\a");
						}
						bool flag7 = false;
						SegmentObject segmentObject7 = this.Editor.Document.SectionProvider.Find(ContentType.ADP, num2 + num4 + 1);
						if (segmentObject7 is DummyObject)
						{
							segmentObject7 = this.Editor.Document.SectionProvider.Find(ContentType.SEC, num2 + num4 + 1);
							if (segmentObject7 is SectionObject)
							{
								flag7 = true;
							}
						}
						if (flag7 && PreferenceManager.Current.Settings.DiffWithoutSection)
						{
							goto IL_0DC7;
						}
						if (PreferenceManager.Current.Settings.DiffWithANSI && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "UNIQUE", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "DISTINCT", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "sysdate", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "CURRENT_DATE", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "TRUNC(SYSTIMESTAMP)", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "SYSTIMESTAMP", RegexOptions.IgnoreCase) && !Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "CURRENT_TIMESTAMP", RegexOptions.IgnoreCase))
						{
							num3++;
							num2++;
						}
						else
						{
							if (Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "^\\s*{<") || string.IsNullOrEmpty(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", "")))
							{
								goto IL_0DC7;
							}
							if (!Regex.IsMatch(((TextLine)diffList_TextFile2.GetByIndex(num2)).Line.Replace("\a", ""), "^\\s*#"))
							{
								if (!PreferenceManager.Current.Settings.SimplifyColor)
								{
									this.Diff_Editor.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.AddDestination);
								}
								flag = true;
								goto IL_0DC7;
							}
							if (PreferenceManager.Current.Settings.DiffComment)
							{
								if (!PreferenceManager.Current.Settings.SimplifyColor)
								{
									this.Diff_Editor.AddRenderRegion(num2 + 1 + num4, DiffResultSpanStatus.AddDestination);
								}
								flag = true;
								goto IL_0DC7;
							}
							goto IL_0DC7;
						}
						IL_0DF9:
						l++;
						continue;
						IL_0DC7:
						if (flag)
						{
							DiffColor diffColor3 = new DiffColor();
							diffColor3.LineNumber = num + 1 + num3;
							this._diffColorAreaList.Add(diffColor3);
						}
						num3++;
						num2++;
						goto IL_0DF9;
					}
					break;
				}
				}
			}
			arrayList.Clear();
			this.Diff_Editor.Text = stringBuilder.ToString();
			ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffColorArea = this.DiffColorAreaList;
			ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffTotalLineLumber = this.Editor.Document.LineCount;
			if (PreferenceManager.Current.Settings.SimplifyColor)
			{
				HighlightingDiffColorize highlightingDiffColorize = new HighlightingDiffColorize();
				highlightingDiffColorize.DiffColorlist = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffColorArea;
				this.Editor.TextArea.TextView.LineTransformers.Add(highlightingDiffColorize);
				this.Diff_Editor.TextArea.TextView.LineTransformers.Add(highlightingDiffColorize);
			}
			this.Diff_Editor.DiffCopy = this.DiffColorAreaList;
			List<ADPModel> customs = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffModel.Customs;
			bool flag8 = true;
			if (customs != null)
			{
				foreach (ADPModel adpmodel in customs)
				{
					flag8 = true;
					string name = adpmodel.Name;
					string name2;
					if ((name2 = name) == null || !(name2 == "global.memo"))
					{
						AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.Name == name).ElementAtOrDefault<AddPointModel>(0);
						if (addPointModel != null)
						{
							SegmentObject segmentObject8 = this.Editor.Document.SectionProvider.Find(addPointModel.ID);
							if (!(null == segmentObject8))
							{
								foreach (DiffColor diffColor4 in this.DiffColorAreaList)
								{
									if (diffColor4.LineNumber >= segmentObject8.Offset && diffColor4.LineNumber <= segmentObject8.EndOffset)
									{
										flag8 = false;
										break;
									}
								}
								adpmodel.CusToStd = flag8;
							}
						}
					}
				}
			}
		}

		// Token: 0x0600029F RID: 671 RVA: 0x00016920 File Offset: 0x00014B20
		private void Load(LoadInformation option)
		{
			if (this.ProgramKey != option.key)
			{
				return;
			}
			this.View.OnPropertyChanged("ProgramKey");
			EventController.GetInstance().GetEvent<LoadedSettingEvent>().Unsubscribe(new Action<LoadInformation>(this.Load));
			this.Load(option.key);
			this.Editor.Document.SectionProvider.Attached();
			if (option.IsDiff)
			{
				this.View.DiffInit();
				this.Diff_Editor = this.View.textDiffEditor;
				for (int i = 1; i <= this.Editor.Document.LineCount; i++)
				{
					DocumentLine lineByNumber = this.Editor.Document.GetLineByNumber(i);
					lineByNumber.RealLineNumber = i;
				}
				this.Diff();
				this.Editor.TextArea.TextView.ScrollOffsetChanged += this.main_ScrollOffsetChanged;
				this._diff_editor.TextArea.TextView.ScrollOffsetChanged += this.diff_ScrollOffsetChanged;
			}
			if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).login_user == "topman" || ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).login_user == "topapp")
			{
				ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsTopstdMode = PreferenceManager.Current.Settings.TopstdEditPermission;
			}
			this.View.DataContext = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey);
			this.Editor.Mode = ContentType.ADP;
			this.Editor.Document.UndoStack.ClearAll();
			string infoXML = SettingManager.Get().GetTzpManger(this.ProgramKey).infoXML;
			if (SettingManager.Get().GetTzpManger(this.ProgramKey).IsMajorAbnormal && !string.IsNullOrEmpty(infoXML))
			{
				SearchKeywordEventArgs e = new SearchKeywordEventArgs(this.ProgramKey, infoXML, DiffType.Normal);
				EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Publish(e);
				SearchKeywordEventArgs e2 = new SearchKeywordEventArgs(this.ProgramKey, infoXML, DiffType.Diff);
				EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Publish(e2);
				NormalizationSearchResult currentSearchResult = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).CurrentSearchResult;
				if (currentSearchResult != null)
				{
					this.Diff_Editor.CaretOffset = currentSearchResult.StartOffset;
				}
			}
		}

		// Token: 0x060002A0 RID: 672 RVA: 0x00016B88 File Offset: 0x00014D88
		private void Load(PackageKey key)
		{
			this.SubscribeEvent();
			this.LoadContent();
			ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).FullContext = this.Editor.Text;
			ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).RefreshTreeNodes();
			IEnumerable<AddPointModel> enumerable = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel model) => model.IsMarkHard && model.IsMarkable == YesNo.Y);
			foreach (AddPointModel addPointModel in enumerable)
			{
				SegmentObject segmentObject = this.Editor.Document.SectionProvider.Find(addPointModel.ID);
				this.Editor.AddMarkedArea(this.Editor.Document.GetLineByNumber(segmentObject.Offset).Offset);
			}
			this.Editor.IsContentLoaded = true;
		}

		// Token: 0x060002A1 RID: 673 RVA: 0x00016C90 File Offset: 0x00014E90
		private void LoadContent()
		{
			this.Editor.Document.SectionProvider.Dettached();
			StringBuilder stringBuilder = new StringBuilder(ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).TGL);
			this.ProcessFunctionTypes(DefinitionType.FUNCTION, ref stringBuilder);
			this.ProcessFunctionTypes(DefinitionType.DIALOG, ref stringBuilder);
			this.ProcessFunctionTypes(DefinitionType.REPORT, ref stringBuilder);
			this.Editor.Text = stringBuilder.ToString();
			this.ProcessAddPoints();
			this.ProcessSections();
		}

		// Token: 0x060002A2 RID: 674 RVA: 0x00016D6C File Offset: 0x00014F6C
		private void ProcessSections()
		{
			if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Sections == null || ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Sections.Count == 0)
			{
				throw new Exception("Can't Find Sections");
			}
			string text = this.Editor.Text;
			RegexOptions regexOptions = RegexOptions.IgnoreCase | RegexOptions.Multiline;
			MatchCollection matchCollection = Regex.Matches(text, this.SectionStartPattern, regexOptions);
			MatchCollection matchCollection2 = Regex.Matches(text, this.SectionEndPattern, regexOptions);
			if (matchCollection.Count != matchCollection2.Count)
			{
				throw new Exception(Application.Current.FindResource("Message_SessionErrorCantRead") as string);
			}
			for (int i = 0; i < matchCollection.Count; i++)
			{
				Match match = matchCollection[i];
				Match match2 = matchCollection2[i];
				string name = match.Groups["name"].Value;
				SectionModel sectionModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Sections.Where<SectionModel>((SectionModel sec) => sec.Name == name).ElementAtOrDefault<SectionModel>(0);
				if (sectionModel == null)
				{
					IEnumerable<SectionModel> enumerable = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Sections.Where<SectionModel>((SectionModel sec) => sec.Source.Attribute("src").Value == "c");
					if (enumerable.Count<SectionModel>() > 0)
					{
						string text2 = enumerable.ElementAtOrDefault<SectionModel>(0).Source.ToString();
						sectionModel = SectionModel.Clone(this.ProgramKey, text2, name);
						ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Sections.Add(sectionModel);
					}
					else
					{
						IEnumerable<SectionModel> enumerable2 = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Sections.Where<SectionModel>((SectionModel sec) => sec.Source.Attribute("src").Value == "m");
						if (enumerable2.Count<SectionModel>() <= 0)
						{
							throw new Exception(string.Format(Application.Current.FindResource("Message_CantFindSession") as string, name));
						}
						string text3 = enumerable2.ElementAtOrDefault<SectionModel>(0).Source.ToString();
						sectionModel = SectionModel.Clone(this.ProgramKey, text3, name);
						ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Sections.Add(sectionModel);
					}
				}
				int num = this.Editor.Document.GetLineByOffset(match.Index).LineNumber + 1;
				int num2 = this.Editor.Document.GetLineByOffset(match2.Index).LineNumber - 1;
				SectionObject sectionObject = new SectionObject(this.ProgramKey, sectionModel.ID);
				if (sectionModel.Name == this.ProgramKey.Program + ".other_function" || sectionModel.Name == this.ProgramKey.Program + ".other_dialog" || sectionModel.Name == this.ProgramKey.Program + ".other_report")
				{
					sectionModel.IsReadOnly = true;
				}
				else if (Regex.IsMatch(match.Value, this.SectionReadOnlyPattern, RegexOptions.IgnoreCase | RegexOptions.Multiline))
				{
					Match match3 = Regex.Match(match.Value, this.SectionReadOnlyPattern, RegexOptions.IgnoreCase | RegexOptions.Multiline);
					sectionModel.IsReadOnly = match3.Groups["value"].Value.Equals("y", StringComparison.CurrentCultureIgnoreCase);
				}
				sectionObject.Length = num2 - num + 1;
				this.Editor.Document.SectionProvider.AddSection(num, sectionObject);
			}
		}

		// Token: 0x060002A3 RID: 675 RVA: 0x00017144 File Offset: 0x00015344
		private void ProcessFunctionTypes(DefinitionType type, ref StringBuilder content)
		{
			RegexOptions regexOptions = RegexOptions.IgnoreCase | RegexOptions.Multiline;
			Regex regex = new Regex("{<point\\s+name=\"(other.function)\".*\\s*/>}", regexOptions);
			if (type == DefinitionType.FUNCTION)
			{
				regex = new Regex("{<point\\s+name=\"(other.function)\".*\\s*/>}", regexOptions);
			}
			else if (type == DefinitionType.DIALOG)
			{
				regex = new Regex("{<point\\s+name=\"(other.dialog)\".*\\s*/>}", regexOptions);
			}
			else if (type == DefinitionType.REPORT)
			{
				regex = new Regex("{<point\\s+name=\"(other.report)\".*\\s*/>}", regexOptions);
			}
			if (regex.IsMatch(content.ToString()))
			{
				StringBuilder stringBuilder = new StringBuilder();
				IOrderedEnumerable<AddPointModel> orderedEnumerable = from a in ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints
					where (a.Status & Status.DELETE) == Status.NULL && a.Type == type
					orderby a.SortIndex
					select a;
				foreach (AddPointModel addPointModel in orderedEnumerable)
				{
					if (stringBuilder.Length > 0)
					{
						stringBuilder.Append(Environment.NewLine);
					}
					string text = string.Format("{{<point name=\"{0}\"/>}}", addPointModel.Name);
					stringBuilder.Append(text);
				}
				content = new StringBuilder(regex.Replace(content.ToString(), stringBuilder.ToString()));
			}
		}

		// Token: 0x060002A4 RID: 676 RVA: 0x000172D4 File Offset: 0x000154D4
		private void ProcessAddPoints()
		{
			for (DocumentLine documentLine = this.Editor.Document.GetLineByNumber(1); documentLine != null; documentLine = documentLine.NextLine)
			{
				string text = this.Editor.Document.GetText(documentLine);
				if (this.addPointRex.IsMatch(text))
				{
					Match match = this.addPointRex.Match(text);
					string name = match.Groups[1].Value;
					AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.Name == name && (a.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
					if (addPointModel == null)
					{
						addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Initial(name);
					}
					addPointModel.IsLoaded = true;
					addPointModel.TglTag = match.Value;
					if (this.markRex.IsMatch(text))
					{
						string value = this.markRex.Matches(text)[0].Groups[1].Value;
						if (string.Equals("Y", value, StringComparison.InvariantCultureIgnoreCase))
						{
							addPointModel.IsMarkable = YesNo.Y;
						}
					}
					if (this.envRex.IsMatch(text))
					{
						addPointModel.EditEnv = this.envRex.Matches(text)[0].Groups[1].Value;
					}
					TextSegment textSegment = new TextSegment
					{
						StartOffset = documentLine.Offset,
						EndOffset = documentLine.EndOffset
					};
					this.Editor.Document.Replace(textSegment, addPointModel.ToString());
					try
					{
						EditObject editObject = this.ProcessEditableArea(addPointModel, documentLine.LineNumber);
						if (editObject != null)
						{
							documentLine = this.Editor.TextArea.Document.GetLineByNumber(editObject.EndOffset);
						}
					}
					catch (DuplicateNameException)
					{
						throw new Exception(string.Format(Application.Current.FindResource("Message_ADPNameExist") as string, name));
					}
				}
			}
		}

		// Token: 0x060002A5 RID: 677 RVA: 0x000174E8 File Offset: 0x000156E8
		private EditObject ProcessEditableArea(AddPointModel model, int startLineNumber)
		{
			string text = model.ToString();
			EditObject editObject = new EditObject(this.ProgramKey, model.ID);
			int num = text.Count<char>((char c) => c == '\n') + 1;
			this.Editor.TextArea.Document.SectionProvider.AddDeletableSegments(startLineNumber, editObject);
			if (model.IsSelfDefinition)
			{
				ParseResult parseResult = FglParserQuickHelper.ParseFunction(model.ToString());
				if (parseResult == null)
				{
					throw new FormatException(string.Format("TAP：'{0}' has format error", model.Name));
				}
				int num2 = startLineNumber + parseResult.StartLineNumber;
				int num3 = num - parseResult.StartLineNumber - 1;
				int num4 = startLineNumber + num - 1;
				editObject.AddChild(model.ID, num2, num3, num4);
			}
			else
			{
				editObject.Length = num;
			}
			return editObject;
		}

		// Token: 0x060002A6 RID: 678 RVA: 0x000175B6 File Offset: 0x000157B6
		private void OnTzpFileClose(PackageKey key)
		{
			if (this.ProgramKey == key)
			{
				this.Dispose();
			}
		}

		// Token: 0x060002A7 RID: 679 RVA: 0x000175CC File Offset: 0x000157CC
		private void OnFieldSelected(FieldArgs args)
		{
			if (args.ProgramKey != this.ProgramKey)
			{
				return;
			}
			string text = string.Format("AFTER FIELD {0}", args.Field);
			IEnumerable<TreeItem> enumerable = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Find(text);
			foreach (TreeItem treeItem in enumerable)
			{
				if (treeItem.Parent.Name.StartsWith("input array", StringComparison.InvariantCultureIgnoreCase) || treeItem.Parent.Name.StartsWith("input by name", StringComparison.InvariantCultureIgnoreCase))
				{
					treeItem.PublishSelectedEvent();
					treeItem.IsExpanded = true;
					break;
				}
			}
		}

		// Token: 0x060002A8 RID: 680 RVA: 0x00017688 File Offset: 0x00015888
		private void OnClosedSearchBox(string empty)
		{
			this.Editor.ClearSearchResult();
		}

		// Token: 0x060002A9 RID: 681 RVA: 0x00017698 File Offset: 0x00015898
		private void OnFunctionSelected(FunctionSelectedModel model)
		{
			if (model.ProgramKey != this.ProgramKey)
			{
				return;
			}
			if (model == null)
			{
				this.Editor.TextArea.ScrollTo(1);
				return;
			}
			if (model.Child != null)
			{
				this.SearchContent(model, 0);
				return;
			}
			this.SearchBlock(model);
		}

		// Token: 0x060002AA RID: 682 RVA: 0x00017714 File Offset: 0x00015914
		private void CreateFunction(CreateFunctionContentArgs info)
		{
			if (this.ProgramKey.Program != info.ProgramKey.Program)
			{
				return;
			}
			AddPointModel model = new AddPointModel(this.ProgramKey);
			model.Content = info.Content;
			model.Status = Status.CREATE;
			model.Ind_fun = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Topind;
			IEnumerable<AddPointModel> enumerable = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => (a.Status & Status.DELETE) != Status.DELETE && a.Name == model.Name);
			if (enumerable.Count<AddPointModel>() > 0)
			{
				throw new Exception(Application.Current.FindResource("Message_NameAlreadyExist") as string);
			}
			DocumentLine documentLine = this.Editor.Document.GetLineByNumber(this.Editor.LineCount);
			this.Editor.Document.UndoStack.StartUndoGroup();
			this.Editor.Document.Insert(documentLine.EndOffset, Environment.NewLine);
			documentLine = this.Editor.Document.GetLineByNumber(this.Editor.LineCount);
			TextSegment textSegment = new TextSegment
			{
				StartOffset = documentLine.Offset,
				EndOffset = documentLine.EndOffset
			};
			this.Editor.Document.Replace(textSegment, model.ToString());
			this.Editor.Document.UndoStack.EndUndoGroup();
		}

		// Token: 0x060002AB RID: 683 RVA: 0x000178A0 File Offset: 0x00015AA0
		private void OnCreateFunctionModel(CreateFunctionInformation info)
		{
			if (!this.Editor.Document.IsInUpdate)
			{
				return;
			}
			if (this.Editor.Document.UndoStack.IsPause)
			{
				return;
			}
			if (!this.Editor.IsContentLoaded)
			{
				return;
			}
			if (string.IsNullOrWhiteSpace(info.Content))
			{
				return;
			}
			AddPointModel addPointModel = new AddPointModel(this.ProgramKey);
			addPointModel.Content = info.Content;
			addPointModel.Status = Status.CREATE;
			addPointModel.IsLoaded = true;
			addPointModel.Ind_fun = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Topind;
			this.ProcessEditableArea(addPointModel, this.Editor.Document.GetLineByOffset(info.Offset).LineNumber);
			ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Add(addPointModel);
		}

		// Token: 0x060002AC RID: 684 RVA: 0x000179A0 File Offset: 0x00015BA0
		private void OnModifyFunctionModel(ModifyFunctionInformation info)
		{
			if (!this.Editor.Document.IsInUpdate)
			{
				return;
			}
			if (info.InsertContent == null)
			{
				return;
			}
			AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.ID == info.ID && (a.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
			if (addPointModel != null && addPointModel.IsSelfDefinition)
			{
				AddPointModel addPointModel2 = addPointModel.Clone();
				addPointModel2.Content = info.Text.Replace("\a\r\n", "").Replace("\n\a", "").Replace("\a\n", "")
					.Replace("\a", "");
				if (addPointModel.Name != addPointModel2.Name || addPointModel.Description.Replace("\r\n", "\n") != addPointModel2.Description.Replace("\a\n", "") || addPointModel.Scope != addPointModel2.Scope)
				{
					info.ProgramKey = this.ProgramKey;
					info.Source = addPointModel2.Name;
					info.Modified = addPointModel2.FunctionName;
					info.Scope = addPointModel2.Scope;
					info.Description = addPointModel2.Description;
					ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).FunctionModify(info);
				}
			}
		}

		// Token: 0x060002AD RID: 685 RVA: 0x00017B5C File Offset: 0x00015D5C
		private void DeleteFunction(string name)
		{
			if (this.ProgramKey != Application.Current.MainWindow.Tag as PackageKey)
			{
				return;
			}
			try
			{
				AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel a) => a.Name == name && (a.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
				SegmentObject segmentObject = this.Editor.Document.SectionProvider.Find(addPointModel.ID);
				if (null == segmentObject)
				{
					throw new Exception(string.Format("找不到{0}", addPointModel.Name));
				}
				SegmentObject nextNode = segmentObject.NextNode;
				int offset = this.Editor.Document.GetLineByNumber(segmentObject.Offset).Offset;
				int num = ((null == nextNode) ? this.Editor.Document.GetLineByNumber(segmentObject.EndOffset).EndOffset : this.Editor.Document.GetLineByNumber(nextNode.Offset).Offset);
				this.Editor.Document.Remove(offset, num - offset);
				if (SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff)
				{
					EventAggregatorManager.Global.GetEvent<RefreshScreen>().Publish(this.ProgramKey);
				}
			}
			catch
			{
				DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_DelteFailed") as string, name), Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK);
			}
		}

		// Token: 0x060002AE RID: 686 RVA: 0x00017D84 File Offset: 0x00015F84
		private void ModifyFunction(ModifyInfomation info)
		{
			if (info.ProgramKey != this.ProgramKey)
			{
				return;
			}
			AddPointModel addPointModel;
			if ((info.ModifyType & ModifyInfomation.ModifyTypeEnum.Name) == ModifyInfomation.ModifyTypeEnum.Name)
			{
				string[] array = info.Source.Split(new char[] { '.' });
				DefinitionType Type = DefinitionType.FUNCTION;
				string text;
				if ((text = array[0]) != null)
				{
					if (!(text == "function"))
					{
						if (!(text == "dialog"))
						{
							if (text == "report")
							{
								Type = DefinitionType.REPORT;
							}
						}
						else
						{
							Type = DefinitionType.DIALOG;
						}
					}
					else
					{
						Type = DefinitionType.FUNCTION;
					}
				}
				addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel m) => m.FunctionName == info.Modified && (m.Status & Status.DELETE) == Status.NULL && m.Type == Type).ElementAtOrDefault<AddPointModel>(0);
			}
			else
			{
				addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel m) => m.Name == info.Source && (m.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
			}
			SegmentObject segmentObject = this.Editor.Document.SectionProvider.Find(addPointModel.ID);
			if (null == segmentObject)
			{
				throw new Exception(string.Format(Application.Current.FindResource("Message_FunctionNotFound") as string, addPointModel.Name));
			}
			TextSegment textSegment = new TextSegment();
			textSegment.StartOffset = this.Editor.Document.GetLineByNumber(segmentObject.Offset).Offset;
			textSegment.EndOffset = this.Editor.Document.GetLineByNumber(segmentObject.EndOffset).EndOffset;
			this.Editor.Document.Replace(textSegment, addPointModel.ToString());
		}

		// Token: 0x060002AF RID: 687 RVA: 0x00017F68 File Offset: 0x00016168
		private void SearchContent(FunctionSelectedModel model, int startOffset)
		{
			startOffset = ((startOffset < 0) ? 0 : startOffset);
			int num = this.SearchText(startOffset, model.Target, false);
			if (num < 0)
			{
				return;
			}
			int num2 = num;
			if (model.Child == null)
			{
				if (num2 < 0)
				{
					num2 = 0;
				}
				int lineNumber = this.Editor.Document.GetLineByOffset(num2).LineNumber;
				this.Editor.ScrollTo(lineNumber);
				this.Editor.CaretOffset = num2;
				return;
			}
			if (model.Child != null)
			{
				num2 = ((num2 < 0) ? 0 : num2);
				int num3 = this.Editor.Document.GetLineByOffset(num2).EndOffset + 1;
				this.SearchContent(model.Child, num3);
			}
		}

		// Token: 0x060002B0 RID: 688 RVA: 0x0001800C File Offset: 0x0001620C
		internal void SearchBlock(FunctionSelectedModel model)
		{
			int num = this.SearchText(0, model.Target, true);
			if (num < 0)
			{
				num = 1;
			}
			TextArea textArea = this.Editor.TextArea;
			int lineNumber = textArea.Document.GetLineByOffset(num).LineNumber;
			this.Editor.ScrollTo(lineNumber);
		}

		// Token: 0x060002B1 RID: 689 RVA: 0x00018058 File Offset: 0x00016258
		private int SearchText(int startOffset, string target, bool searchComment)
		{
			string text = this.Editor.Document.GetText(startOffset, this.Editor.Document.TextLength - startOffset);
			string text2 = target.Replace(",", ",\\s*");
			if (text2.Contains('('))
			{
				text2 = string.Format("^(?:PUBLIC\\s+|PRIVATE\\s+|)(?:{0})", text2.Replace("(", "\\("));
			}
			else if (text2 == "MAIN")
			{
				text2 = string.Format("^(?:{0})", text2);
			}
			MatchCollection matchCollection = Regex.Matches(text, text2, RegexOptions.IgnoreCase | RegexOptions.Multiline);
			if (matchCollection.Count > 0)
			{
				foreach (object obj in matchCollection)
				{
					Match match = (Match)obj;
					int num = match.Index + startOffset;
					if (searchComment)
					{
						return num;
					}
					DocumentLine lineByOffset = this.Editor.Document.GetLineByOffset(num);
					string text3 = this.Editor.Document.GetText(lineByOffset.Offset, lineByOffset.Length);
					int num2 = text3.IndexOf('#');
					if (num2 <= -1 || num2 + lineByOffset.Offset > num)
					{
						return num;
					}
				}
				return -1;
			}
			return -1;
		}

		// Token: 0x060002B2 RID: 690 RVA: 0x000181A8 File Offset: 0x000163A8
		public void OnFullTextRefreshScreen(PackageKey key)
		{
			this.OnRefreshScreen(key, false);
		}

		// Token: 0x060002B3 RID: 691 RVA: 0x000181B2 File Offset: 0x000163B2
		public void OnRefreshScreen(PackageKey key)
		{
			this.OnRefreshScreen(key, true);
		}

		// Token: 0x060002B4 RID: 692 RVA: 0x000181D4 File Offset: 0x000163D4
		public void OnRefreshScreen(PackageKey key, bool isSave)
		{
			if (this.ProgramKey != key)
			{
				return;
			}
			int caretOffset = this.Editor.CaretOffset;
			ContentType mode = this.Editor.Mode;
			if (isSave)
			{
				this.SaveFile(this.ProgramKey);
			}
			this.Editor.Document.UndoStack.Pause();
			this.LoadContent();
			this.Editor.Document.SectionProvider.Attached();
			this.Editor.Document.UndoStack.ClearAll();
			this.Editor.ClearMarkArea();
			IEnumerable<AddPointModel> enumerable = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel model) => model.IsMarkHard && model.IsMarkable == YesNo.Y);
			foreach (AddPointModel addPointModel in enumerable)
			{
				SegmentObject segmentObject = this.Editor.Document.SectionProvider.Find(addPointModel.ID);
				this.Editor.AddMarkedArea(this.Editor.Document.GetLineByNumber(segmentObject.Offset).Offset);
			}
			if (SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff)
			{
				this.Diff();
				if (mode == ContentType.ADP)
				{
					this.Editor.Mode = ContentType.ADP;
				}
				else if (mode == ContentType.SEC)
				{
					this.Editor.Mode = ContentType.SEC;
				}
			}
			if (this.Editor.Document.UndoStack.IsPause)
			{
				this.Editor.Document.UndoStack.Resume();
			}
			this.Editor.Document.UndoStack.ClearAll();
			this.Editor.CaretOffset = Math.Min(caretOffset, this.Editor.Document.TextLength - 1);
			string infoXML = SettingManager.Get().GetTzpManger(this.ProgramKey).infoXML;
			if (SettingManager.Get().GetTzpManger(this.ProgramKey).IsMajorAbnormal && !string.IsNullOrEmpty(infoXML))
			{
				SearchKeywordEventArgs e = new SearchKeywordEventArgs(this.ProgramKey, infoXML, DiffType.Normal);
				EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Publish(e);
				SearchKeywordEventArgs e2 = new SearchKeywordEventArgs(this.ProgramKey, infoXML, DiffType.Diff);
				EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Publish(e2);
			}
		}

		// Token: 0x060002B5 RID: 693 RVA: 0x00018438 File Offset: 0x00016638
		private void OnRefreshDiffResult(PackageKey key)
		{
			ContentType mode = this.Editor.Mode;
			if (!SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff)
			{
				return;
			}
			if (this.ProgramKey != key)
			{
				return;
			}
			this.OnRefreshScreen(this.ProgramKey);
		}

		// Token: 0x060002B6 RID: 694 RVA: 0x00018484 File Offset: 0x00016684
		private void OnToggleDiffEditable(PackageKey key)
		{
			if (key == this.ProgramKey)
			{
				bool flag = this.Diff_Editor.Mode == ContentType.READONLY;
				if (this.Diff_Editor != null)
				{
					this.Diff_Editor.Document.UndoStack.ClearAll();
					this.Diff_Editor.Mode = (flag ? ContentType.OPEN : ContentType.READONLY);
					this.Diff_Editor.IsChangedMode = true;
				}
				this.Editor.Mode = (flag ? ContentType.READONLY : ContentType.ADP);
			}
		}

		// Token: 0x060002B7 RID: 695 RVA: 0x00018500 File Offset: 0x00016700
		public void Dispose()
		{
			this.UnsubscribeEvent();
			if (this.View != null)
			{
				this.View.Dispose();
			}
			if (this.Editor != null)
			{
				this.Editor.TextChanged -= this.Editor_TextChanged;
				this.Editor.TextArea.TextView.ScrollOffsetChanged -= this.main_ScrollOffsetChanged;
				this.Editor.Dispose();
				this.Editor = null;
			}
			if (this._diff_editor != null)
			{
				this._diff_editor.TextArea.TextView.ScrollOffsetChanged -= this.diff_ScrollOffsetChanged;
				this._diff_editor.Dispose();
			}
		}

		// Token: 0x17000064 RID: 100
		// (get) Token: 0x060002B8 RID: 696 RVA: 0x000185AC File Offset: 0x000167AC
		// (set) Token: 0x060002B9 RID: 697 RVA: 0x000185B4 File Offset: 0x000167B4
		public bool isScrollSynced { get; set; }

		// Token: 0x04000120 RID: 288
		private RelayCommand _showParametersCommand;

		// Token: 0x04000121 RID: 289
		private CodeEditorMainWindow view;

		// Token: 0x04000122 RID: 290
		private CodeTextEditor _editor;

		// Token: 0x04000123 RID: 291
		private DiffTextViewer _diff_editor;

		// Token: 0x04000124 RID: 292
		private List<Guid> _DiffSectionList = new List<Guid>();

		// Token: 0x04000125 RID: 293
		private readonly string SectionStartPattern = "({<section\\s+id=\"(?<name>\\S*)\".*>})";

		// Token: 0x04000126 RID: 294
		private readonly string SectionEndPattern = "({</section>})";

		// Token: 0x04000127 RID: 295
		private readonly string SectionReadOnlyPattern = "readonly=\"(?<value>\\w)\"";

		// Token: 0x04000128 RID: 296
		private List<DiffColor> _diffColorAreaList = new List<DiffColor>();

		// Token: 0x04000129 RID: 297
		public Dictionary<string, string> Diff_List = new Dictionary<string, string>();

		// Token: 0x0400012A RID: 298
		private readonly Regex addPointRex = new Regex("{<point\\s+name=\"(\\S+)\".*\\s*/>}");

		// Token: 0x0400012B RID: 299
		private readonly Regex markRex = new Regex("mark=\"(?<value>\\w)");

		// Token: 0x0400012C RID: 300
		private readonly Regex envRex = new Regex("edit=\"(?<value>\\w)");
	}
}
