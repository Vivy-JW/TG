using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using System.Xml.Linq;
using ICSharpCode.AvalonEdit;
using ICSharpCode.AvalonEdit.Bookmark;
using ICSharpCode.AvalonEdit.Document;
using ICSharpCode.AvalonEdit.Vi;
using SpecDesigner.CodeEditWindow.Helper;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Event;
using SpecDesigner.Infrastructure.Helper;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;
using SpecDesignerCommon.Bookmark;
using SpecDesignerCommon.Events;
using SpecDesignerPreference;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000038 RID: 56
	public partial class CodeEditorMainWindow : UserControl, INotifyPropertyChanged
	{
		// Token: 0x17000055 RID: 85
		// (get) Token: 0x06000239 RID: 569 RVA: 0x00011340 File Offset: 0x0000F540
		// (set) Token: 0x0600023A RID: 570 RVA: 0x00011348 File Offset: 0x0000F548
		public CodeTextEditor textEditor { get; private set; }

		// Token: 0x17000056 RID: 86
		// (get) Token: 0x0600023B RID: 571 RVA: 0x00011351 File Offset: 0x0000F551
		// (set) Token: 0x0600023C RID: 572 RVA: 0x00011359 File Offset: 0x0000F559
		public DiffTextViewer textDiffEditor { get; private set; }

		// Token: 0x17000057 RID: 87
		// (get) Token: 0x0600023D RID: 573 RVA: 0x00011362 File Offset: 0x0000F562
		// (set) Token: 0x0600023E RID: 574 RVA: 0x0001136A File Offset: 0x0000F56A
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x0600023F RID: 575 RVA: 0x00011374 File Offset: 0x0000F574
		private CodeEditorMainWindow()
		{
			this.InitializeComponent();
			ResourceController.GetInstance().Subscribe();
			base.CommandBindings.Add(new CommandBinding(BookmarkCommands.ToggleBookmarkCommand, new ExecutedRoutedEventHandler(this.OnToggleBookmark)));
			base.CommandBindings.Add(new CommandBinding(BookmarkCommands.ClearBookmarkCommand, new ExecutedRoutedEventHandler(this.OnClearBookmark), new CanExecuteRoutedEventHandler(this.CanClearBookmark)));
			base.CommandBindings.Add(new CommandBinding(BookmarkCommands.PreviousBookmarkCommand, new ExecutedRoutedEventHandler(this.OnPreviousBookmark), new CanExecuteRoutedEventHandler(this.CanPreviousBookmark)));
			base.CommandBindings.Add(new CommandBinding(BookmarkCommands.NextBookmarkCommand, new ExecutedRoutedEventHandler(this.OnNextBookmark), new CanExecuteRoutedEventHandler(this.CanNextBookmark)));
			base.CommandBindings.Add(new CommandBinding(BookmarkCommands.DeleteBookmarkCommand, new ExecutedRoutedEventHandler(this.OnDeleteBookmark), new CanExecuteRoutedEventHandler(this.CanDeleteBookmark)));
			base.CommandBindings.Add(new CommandBinding(DiffCommands.ToNextDiffLineCommand, new ExecutedRoutedEventHandler(this.OnToNextDiffLine), new CanExecuteRoutedEventHandler(this.CanToNextDiffLine)));
			base.CommandBindings.Add(new CommandBinding(DiffCommands.ToPreviousDiffLineCommand, new ExecutedRoutedEventHandler(this.OnToPreviousDiffLine), new CanExecuteRoutedEventHandler(this.CanToPreviousDiffLine)));
			this.SwichToTopstd.DataContext = SettingManager.Get().TopstdSetting;
		}

		// Token: 0x17000058 RID: 88
		// (get) Token: 0x06000240 RID: 576 RVA: 0x000114E2 File Offset: 0x0000F6E2
		private IBookmarkMargin iBookmarkMargin
		{
			get
			{
				return this.textEditor.TextArea.TextView.Services.GetService(typeof(IBookmarkMargin)) as IBookmarkMargin;
			}
		}

		// Token: 0x06000241 RID: 577 RVA: 0x00011510 File Offset: 0x0000F710
		private void CanDeleteBookmark(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = true;
			if (this.iBookmarkMargin == null)
			{
				flag = false;
			}
			else if (this.iBookmarkMargin.SelectedBookmark == null)
			{
				flag = false;
			}
			e.CanExecute = flag;
		}

		// Token: 0x06000242 RID: 578 RVA: 0x00011541 File Offset: 0x0000F741
		private void OnDeleteBookmark(object sender, ExecutedRoutedEventArgs e)
		{
			this.iBookmarkMargin.Bookmarks.Remove(this.iBookmarkMargin.SelectedBookmark);
		}

		// Token: 0x06000243 RID: 579 RVA: 0x00011560 File Offset: 0x0000F760
		private void CanNextBookmark(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag;
			if (this.iBookmarkMargin == null || this.iBookmarkMargin.SelectedBookmark == null)
			{
				flag = false;
			}
			else
			{
				int num = this.iBookmarkMargin.Bookmarks.IndexOf(this.iBookmarkMargin.SelectedBookmark);
				flag = num != this.iBookmarkMargin.Bookmarks.Count - 1;
			}
			e.CanExecute = flag;
		}

		// Token: 0x06000244 RID: 580 RVA: 0x000115C4 File Offset: 0x0000F7C4
		private void OnNextBookmark(object sender, ExecutedRoutedEventArgs e)
		{
			int num = this.iBookmarkMargin.Bookmarks.IndexOf(this.iBookmarkMargin.SelectedBookmark);
			this.iBookmarkMargin.SelectedBookmark = this.iBookmarkMargin.Bookmarks[num + 1];
			this.textEditor.ScrollTo(this.iBookmarkMargin.SelectedBookmark.LineNumber);
		}

		// Token: 0x06000245 RID: 581 RVA: 0x00011628 File Offset: 0x0000F828
		private void CanPreviousBookmark(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag;
			if (this.iBookmarkMargin == null || this.iBookmarkMargin.SelectedBookmark == null)
			{
				flag = false;
			}
			else
			{
				int num = this.iBookmarkMargin.Bookmarks.IndexOf(this.iBookmarkMargin.SelectedBookmark);
				flag = num != 0;
			}
			e.CanExecute = flag;
		}

		// Token: 0x06000246 RID: 582 RVA: 0x0001167C File Offset: 0x0000F87C
		private void OnPreviousBookmark(object sender, ExecutedRoutedEventArgs e)
		{
			int num = this.iBookmarkMargin.Bookmarks.IndexOf(this.iBookmarkMargin.SelectedBookmark);
			this.iBookmarkMargin.SelectedBookmark = this.iBookmarkMargin.Bookmarks[num - 1];
			this.textEditor.ScrollTo(this.iBookmarkMargin.SelectedBookmark.LineNumber);
		}

		// Token: 0x06000247 RID: 583 RVA: 0x000116E0 File Offset: 0x0000F8E0
		private void CanSelectBookmark(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = true;
			if (this.iBookmarkMargin == null || this.iBookmarkMargin.SelectedBookmark == null)
			{
				flag = false;
			}
			e.CanExecute = flag;
		}

		// Token: 0x06000248 RID: 584 RVA: 0x00011710 File Offset: 0x0000F910
		private void OnSelectBookmark(object sender, ExecutedRoutedEventArgs e)
		{
			if (this.iBookmarkMargin == null || this.iBookmarkMargin.SelectedBookmark == null)
			{
				return;
			}
			int lineNumber = this.textEditor.Document.GetLineByOffset(this.textEditor.CaretOffset).LineNumber;
			if (this.iBookmarkMargin.SelectedBookmark.LineNumber == lineNumber)
			{
				return;
			}
			this.textEditor.ScrollTo(this.iBookmarkMargin.SelectedBookmark.LineNumber);
		}

		// Token: 0x06000249 RID: 585 RVA: 0x0001179C File Offset: 0x0000F99C
		private void OnToggleBookmark(object sender, ExecutedRoutedEventArgs e)
		{
			if (this.iBookmarkMargin != null)
			{
				int lineNumber = this.textEditor.Document.GetLineByOffset(this.textEditor.CaretOffset).LineNumber;
				IBookmark bookmark = this.iBookmarkMargin.Bookmarks.Where<IBookmark>((IBookmark bm) => bm.LineNumber == lineNumber).ElementAtOrDefault<IBookmark>(0);
				if (bookmark != null)
				{
					this.iBookmarkMargin.Bookmarks.Remove(bookmark);
					return;
				}
				TextLocation location = this.textEditor.Document.GetLocation(this.textEditor.Document.GetOffset(lineNumber, 0));
				BookmarkBase bookmarkBase = new BookmarkBase(location)
				{
					BookmarkMargin = this.iBookmarkMargin
				};
				bookmarkBase.Document = this.textEditor.Document;
				this.iBookmarkMargin.Bookmarks.Add(bookmarkBase);
			}
		}

		// Token: 0x0600024A RID: 586 RVA: 0x0001187C File Offset: 0x0000FA7C
		private void CanClearBookmark(object sender, CanExecuteRoutedEventArgs e)
		{
			IBookmarkMargin bookmarkMargin = this.textEditor.TextArea.TextView.Services.GetService(typeof(IBookmarkMargin)) as IBookmarkMargin;
			e.CanExecute = bookmarkMargin != null && bookmarkMargin.Bookmarks.Count > 0;
		}

		// Token: 0x0600024B RID: 587 RVA: 0x000118D0 File Offset: 0x0000FAD0
		private void OnClearBookmark(object sender, ExecutedRoutedEventArgs e)
		{
			IBookmarkMargin bookmarkMargin = this.textEditor.TextArea.TextView.Services.GetService(typeof(IBookmarkMargin)) as IBookmarkMargin;
			if (bookmarkMargin != null)
			{
				bookmarkMargin.Bookmarks.Clear();
			}
		}

		// Token: 0x0600024C RID: 588 RVA: 0x00011938 File Offset: 0x0000FB38
		public CodeEditorMainWindow(PackageKey key)
			: this()
		{
			this.ProgramKey = key;
			this.textEditor = new CodeTextEditor(this.ProgramKey);
			this._codeEditorManager = new CodeEditorManager(this, key);
			this._structureHelper = new StructureHelper(this.ProgramKey, this.textEditor);
			this.treeView.Load(this.ProgramKey);
			if (!SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff)
			{
				this.dockPanel.Children.Add(this.textEditor);
				this.aptGrid.Visibility = Visibility.Collapsed;
			}
			else
			{
				string ColumnHeader = Application.Current.FindResource("CE_SuggestReturnToStandard") as string;
				DataGridColumn dataGridColumn = this.customDataGrid.Columns.Where<DataGridColumn>((DataGridColumn c) => c.Header.ToString() == ColumnHeader).FirstOrDefault<DataGridColumn>();
				if (!SettingManager.Get().GetTzpManger(this.ProgramKey).IsStandardProgram)
				{
					dataGridColumn.Visibility = Visibility.Collapsed;
				}
				else
				{
					dataGridColumn.Visibility = Visibility.Visible;
				}
			}
			this._viManager = new ViEditMode(this.textEditor.TextArea, this.ProgramKey);
			this.viModeBlock.DataContext = this._viManager;
			base.CommandBindings.Add(new CommandBinding(AvalonEditCommands.EnableViMode, new ExecutedRoutedEventHandler(this.OnEnableViModeChanged)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.CreateFunction, new ExecutedRoutedEventHandler(this.ExecutedCreateFunction), new CanExecuteRoutedEventHandler(this.CanCreateFunction)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.DeleteFunction, new ExecutedRoutedEventHandler(CodeEditorMainWindow.ExecutedDeleteFunction), new CanExecuteRoutedEventHandler(CodeEditorMainWindow.CanDeleteFunction)));
			this.notBookingWatermark.Visibility = (SettingManager.Get().GetTzpManger(this.ProgramKey).Booking ? Visibility.Collapsed : Visibility.Visible);
			if (PreferenceManager.Current.Settings.ViMode)
			{
				this.textEditor.TextArea.Options.EnableViMode = !this.textEditor.TextArea.Options.EnableViMode;
				if (this.textEditor.TextArea.Options.EnableViMode)
				{
					this.viModeBlock.Visibility = Visibility.Visible;
				}
				else
				{
					this.viModeBlock.Visibility = Visibility.Collapsed;
				}
			}
			base.CommandBindings.Add(new CommandBinding(DiffCommands.ToNextNormalizationCommand, new ExecutedRoutedEventHandler(this.OnToNextNormalization), new CanExecuteRoutedEventHandler(this.CanToNextNormalization)));
			base.CommandBindings.Add(new CommandBinding(DiffCommands.ToPreviousNormalizationCommand, new ExecutedRoutedEventHandler(this.OnPreviousNormalization), new CanExecuteRoutedEventHandler(this.CanToPreviousNormalization)));
		}

		// Token: 0x0600024D RID: 589 RVA: 0x00011BDC File Offset: 0x0000FDDC
		public void DiffInit()
		{
			Style style = base.FindResource("VerticalGridSplitterStyle") as Style;
			Style style2 = base.FindResource("HorizontalGridSplitterStyle") as Style;
			this.textDiffEditor = new DiffTextViewer(this.ProgramKey);
			this.textDiffEditor.Text = SettingManager.Get().GetTzpManger(this.ProgramKey).DIFF_SRC;
			if (PreferenceManager.Current.Settings.HorizontalArrangement)
			{
				RowDefinition rowDefinition = new RowDefinition();
				rowDefinition.Height = new GridLength(0.55, GridUnitType.Star);
				RowDefinition rowDefinition2 = new RowDefinition();
				rowDefinition2.Height = new GridLength(3.0, GridUnitType.Auto);
				RowDefinition rowDefinition3 = new RowDefinition();
				rowDefinition3.Height = new GridLength(0.55, GridUnitType.Star);
				this.dockPanel.RowDefinitions.Add(rowDefinition);
				this.dockPanel.RowDefinitions.Add(rowDefinition2);
				this.dockPanel.RowDefinitions.Add(rowDefinition3);
				this.textEditor.SetValue(Grid.RowProperty, 0);
				this.dockPanel.Children.Add(this.textEditor);
				GridSplitter gridSplitter = new GridSplitter();
				gridSplitter.Style = style2;
				gridSplitter.Height = 10.0;
				gridSplitter.VerticalAlignment = VerticalAlignment.Center;
				gridSplitter.ShowsPreview = true;
				gridSplitter.HorizontalAlignment = HorizontalAlignment.Stretch;
				gridSplitter.SetValue(Grid.RowProperty, 1);
				this.dockPanel.Children.Add(gridSplitter);
				this.textDiffEditor.SetValue(Grid.RowProperty, 2);
				this.dockPanel.Children.Add(this.textDiffEditor);
			}
			else
			{
				ColumnDefinition columnDefinition = new ColumnDefinition();
				columnDefinition.Width = new GridLength(0.55, GridUnitType.Star);
				ColumnDefinition columnDefinition2 = new ColumnDefinition();
				columnDefinition2.Width = new GridLength(3.0, GridUnitType.Auto);
				ColumnDefinition columnDefinition3 = new ColumnDefinition();
				columnDefinition3.Width = new GridLength(0.55, GridUnitType.Star);
				this.dockPanel.ColumnDefinitions.Add(columnDefinition);
				this.dockPanel.ColumnDefinitions.Add(columnDefinition2);
				this.dockPanel.ColumnDefinitions.Add(columnDefinition3);
				this.textEditor.SetValue(Grid.ColumnProperty, 2);
				this.dockPanel.Children.Add(this.textEditor);
				GridSplitter gridSplitter2 = new GridSplitter();
				gridSplitter2.Style = style;
				gridSplitter2.Width = 10.0;
				gridSplitter2.VerticalAlignment = VerticalAlignment.Stretch;
				gridSplitter2.ShowsPreview = true;
				gridSplitter2.HorizontalAlignment = HorizontalAlignment.Center;
				gridSplitter2.SetValue(Grid.ColumnProperty, 1);
				this.dockPanel.Children.Add(gridSplitter2);
				this.textDiffEditor.SetValue(Grid.ColumnProperty, 0);
				this.dockPanel.Children.Add(this.textDiffEditor);
			}
			this.mainGrid.ColumnDefinitions.Insert(0, new ColumnDefinition
			{
				Width = new GridLength(0.2, GridUnitType.Star)
			});
			this.treeView.SetValue(Grid.ColumnProperty, 1);
			GridSplitter gridSplitter3 = new GridSplitter();
			gridSplitter3.Style = style;
			gridSplitter3.Name = "treeGridSplitter";
			gridSplitter3.Width = 10.0;
			gridSplitter3.VerticalAlignment = VerticalAlignment.Stretch;
			gridSplitter3.HorizontalAlignment = HorizontalAlignment.Right;
			gridSplitter3.ShowsPreview = true;
			gridSplitter3.SetValue(Grid.ColumnProperty, 1);
			gridSplitter3.SetValue(Grid.RowSpanProperty, 2);
			this.mainGrid.Children.Add(gridSplitter3);
			this._codeEditorManager.isScrollSynced = true;
			if (SettingManager.Get().GetTzpManger(this.ProgramKey).ProgIdentity == "s" && SettingManager.Get().GetTzpManger(this.ProgramKey).IsStandardProgram)
			{
				this.customDataGrid.Visibility = Visibility.Collapsed;
			}
			else
			{
				this.aptGrid.RowDefinitions.ElementAt<RowDefinition>(1).Height = new GridLength(0.55, GridUnitType.Star);
				base.UpdateLayout();
			}
			if (SettingManager.Get().GetTzpManger(this.ProgramKey).IsStandardProgram)
			{
				this.customDataGrid.Columns[2].Header = Application.Current.FindResource("codeEditor_CustomedAddPoint") as string;
			}
			else
			{
				this.customDataGrid.Columns[2].Header = Application.Current.FindResource("codeEditor_UncitedAddPoint") as string;
			}
			if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffModel != null)
			{
				if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffModel.Customs.Count<ADPModel>() <= 0 && ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffModel.Merge.Elements().Count<XElement>() <= 0)
				{
					this.aptGrid.Visibility = Visibility.Hidden;
					this.aptGrid.Width = 0.0;
					this.mainGrid.ColumnDefinitions.ElementAt<ColumnDefinition>(0).Width = new GridLength(0.0, GridUnitType.Auto);
					return;
				}
				this.aptGrid.Visibility = Visibility.Visible;
			}
		}

		// Token: 0x0600024E RID: 590 RVA: 0x00012154 File Offset: 0x00010354
		private void OnToNextDiffLine(object sender, ExecutedRoutedEventArgs e)
		{
			if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffColorArea != null)
			{
				DocumentLine CurrentLine = this.textEditor.Document.GetLineByOffset(this.textEditor.CaretOffset);
				DocumentLine documentLine = null;
				List<DiffColor> diffColorArea = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffColorArea;
				if (diffColorArea.Count<DiffColor>() > 0)
				{
					if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffColorArea.Where<DiffColor>((DiffColor a) => a.LineNumber == CurrentLine.LineNumber).FirstOrDefault<DiffColor>() == null)
					{
						for (int i = 0; i < diffColorArea.Count<DiffColor>(); i++)
						{
							if (diffColorArea[i].LineNumber > CurrentLine.LineNumber)
							{
								documentLine = this.textEditor.Document.GetLineByNumber(diffColorArea[i].LineNumber);
								break;
							}
						}
					}
					else
					{
						for (int j = 0; j < diffColorArea.Count<DiffColor>(); j++)
						{
							if (diffColorArea[j].LineNumber == CurrentLine.LineNumber)
							{
								for (int k = j + 1; k < diffColorArea.Count<DiffColor>(); k++)
								{
									if (diffColorArea[k].LineNumber != diffColorArea[k - 1].LineNumber + 1)
									{
										documentLine = this.textEditor.Document.GetLineByNumber(diffColorArea[k].LineNumber);
										break;
									}
								}
							}
						}
					}
					if (documentLine == null)
					{
						documentLine = this.textEditor.Document.GetLineByNumber(diffColorArea[0].LineNumber);
					}
					this.textEditor.CaretOffset = documentLine.Offset;
					this.textEditor.ScrollTo(documentLine.LineNumber);
				}
			}
		}

		// Token: 0x0600024F RID: 591 RVA: 0x00012322 File Offset: 0x00010522
		private void CanToNextDiffLine(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff && ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffColorArea.Count<DiffColor>() > 0;
		}

		// Token: 0x06000250 RID: 592 RVA: 0x00012380 File Offset: 0x00010580
		private void OnToPreviousDiffLine(object sender, ExecutedRoutedEventArgs e)
		{
			if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffColorArea != null)
			{
				DocumentLine CurrentLine = this.textEditor.Document.GetLineByOffset(this.textEditor.CaretOffset);
				DocumentLine documentLine = null;
				List<DiffColor> diffColorArea = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffColorArea;
				if (diffColorArea.Count<DiffColor>() > 0)
				{
					if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffColorArea.Where<DiffColor>((DiffColor a) => a.LineNumber == CurrentLine.LineNumber).FirstOrDefault<DiffColor>() == null)
					{
						for (int i = diffColorArea.Count<DiffColor>() - 1; i >= 0; i--)
						{
							if (diffColorArea[i].LineNumber < CurrentLine.LineNumber)
							{
								documentLine = this.textEditor.Document.GetLineByNumber(diffColorArea[i].LineNumber);
								break;
							}
						}
					}
					else
					{
						for (int j = 0; j < diffColorArea.Count<DiffColor>(); j++)
						{
							if (diffColorArea[j].LineNumber == CurrentLine.LineNumber)
							{
								for (int k = j; k > 0; k--)
								{
									if (diffColorArea[k].LineNumber - 1 != diffColorArea[k - 1].LineNumber)
									{
										documentLine = this.textEditor.Document.GetLineByNumber(diffColorArea[k - 1].LineNumber);
										break;
									}
								}
							}
						}
					}
					if (documentLine == null)
					{
						documentLine = this.textEditor.Document.GetLineByNumber(diffColorArea[diffColorArea.Count<DiffColor>() - 1].LineNumber);
					}
					this.textEditor.CaretOffset = documentLine.Offset;
					this.textEditor.ScrollTo(documentLine.LineNumber);
				}
			}
		}

		// Token: 0x06000251 RID: 593 RVA: 0x0001254F File Offset: 0x0001074F
		private void CanToPreviousDiffLine(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff && ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).DiffColorArea.Count<DiffColor>() > 0;
		}

		// Token: 0x06000252 RID: 594 RVA: 0x00012590 File Offset: 0x00010790
		private void CanCreateFunction(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			if (this.textEditor.Mode == ContentType.SEC)
			{
				flag = false;
			}
			else
			{
				TreeItem treeItem = e.Parameter as TreeItem;
				if (treeItem != null && treeItem.IsFolder)
				{
					flag = true;
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x06000253 RID: 595 RVA: 0x000125D4 File Offset: 0x000107D4
		private void ExecutedCreateFunction(object sender, ExecutedRoutedEventArgs e)
		{
			FunctionInfoWindow functionInfoWindow = new FunctionInfoWindow();
			functionInfoWindow.Show((e.Parameter as TreeItem).Name);
		}

		// Token: 0x06000254 RID: 596 RVA: 0x0001266C File Offset: 0x0001086C
		public static void CanDeleteFunction(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			AddPointModel addPointModel = null;
			if (sender.GetType() == typeof(CodeEditorMainWindow))
			{
				CodeEditorMainWindow codeEditorMainWindow = sender as CodeEditorMainWindow;
				if (codeEditorMainWindow.textEditor.Mode != ContentType.SEC)
				{
					TreeItem item = e.Parameter as TreeItem;
					if (item != null && !item.IsFolder)
					{
						addPointModel = ResourceController.GetInstance().GetProgramInfo(codeEditorMainWindow.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel m) => m.Name == item.ReferenceName && m.IsNew && (m.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
					}
				}
			}
			else if (sender.GetType() == typeof(CodeTextEditor))
			{
				CodeTextEditor codeTextEditor = sender as CodeTextEditor;
				if (codeTextEditor.Mode != ContentType.SEC)
				{
					SegmentObject segment = codeTextEditor.TextArea.Document.SectionProvider.Find(codeTextEditor.Document.GetLineByOffset(codeTextEditor.CaretOffset).LineNumber);
					if (null != segment)
					{
						addPointModel = ResourceController.GetInstance().GetProgramInfo(codeTextEditor.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel m) => m.ID == segment.ID && m.IsNew && (m.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
					}
				}
			}
			CustomizedRoutedCommand customizedRoutedCommand = e.Command as CustomizedRoutedCommand;
			customizedRoutedCommand.Parameter = addPointModel;
			e.CanExecute = addPointModel != null && addPointModel.IsSelfDefinition && addPointModel.IsNew && addPointModel.IsEditable;
		}

		// Token: 0x06000255 RID: 597 RVA: 0x00012810 File Offset: 0x00010A10
		public static void ExecutedDeleteFunction(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			CustomizedRoutedCommand customizedRoutedCommand = e.Command as CustomizedRoutedCommand;
			AddPointModel addPointModel = customizedRoutedCommand.Parameter as AddPointModel;
			if (MessageBoxResult.Yes == DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_ConfirmDelete") as string, addPointModel.FunctionName), Application.Current.FindResource("Preference_ConfirmDelete") as string, MessageBoxButton.YesNo, MessageBoxImage.Exclamation))
			{
				EventController.GetInstance().GetEvent<DeleteFunctionEvent>().Publish(addPointModel.Name);
				EventAggregatorManager.Get(addPointModel.ProgramKey).GetEvent<CodeChangedEvent>().Publish(addPointModel.ProgramKey);
			}
		}

		// Token: 0x06000256 RID: 598 RVA: 0x000128AC File Offset: 0x00010AAC
		private void OnEnableViModeChanged(object sender, ExecutedRoutedEventArgs e)
		{
			this.textEditor.TextArea.Options.EnableViMode = !this.textEditor.TextArea.Options.EnableViMode;
			if (this.textEditor.TextArea.Options.EnableViMode)
			{
				this.viModeBlock.Visibility = Visibility.Visible;
				return;
			}
			this.viModeBlock.Visibility = Visibility.Collapsed;
		}

		// Token: 0x06000257 RID: 599 RVA: 0x00012916 File Offset: 0x00010B16
		public void Dispose()
		{
			base.CommandBindings.Clear();
			this._structureHelper.Dispose();
		}

		// Token: 0x06000258 RID: 600 RVA: 0x00012930 File Offset: 0x00010B30
		private void checkBox_PreviewMouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			e.Handled = true;
			if (this.textEditor.Mode == ContentType.SEC)
			{
				this._codeEditorManager.SaveSectionContent();
				this.textEditor.Mode = ContentType.ADP;
			}
			else if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsSectionModify)
			{
				this.textEditor.Mode = ContentType.SEC;
			}
			else if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ENV == "s" && ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).login_user != "topstd")
			{
				if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsStdSectionVerify)
				{
					if (MessageBoxResult.Yes == DesignerMessageBox.Show(string.Format("{0}{1}{1}{2}", Application.Current.FindResource("CE_AfterChangeSession") as string, Environment.NewLine, Application.Current.FindResource("Message_PropertyChange") as string), Application.Current.FindResource("Message_Warning") as string, MessageBoxButton.YesNo))
					{
						this.textEditor.Mode = ContentType.SEC;
					}
				}
				else if (MessageBoxResult.OK == DesignerMessageBox.Show(string.Format("{0}{1}{1}{2}", Application.Current.FindResource("Message_SectionVerify") as string, Environment.NewLine, Application.Current.FindResource("Message_SectionVerifyWarning") as string), Application.Current.FindResource("Message_Warning") as string, MessageBoxButton.OK))
				{
					return;
				}
			}
			else if (MessageBoxResult.Yes == DesignerMessageBox.Show(string.Format("{0}{1}{1}{2}", Application.Current.FindResource("CE_AfterChangeSession") as string, Environment.NewLine, Application.Current.FindResource("Message_PropertyChange") as string), Application.Current.FindResource("Message_Warning") as string, MessageBoxButton.YesNo))
			{
				this.textEditor.Mode = ContentType.SEC;
			}
			this.checkBox.IsChecked = new bool?(this.textEditor.Mode == ContentType.SEC);
		}

		// Token: 0x17000059 RID: 89
		// (get) Token: 0x06000259 RID: 601 RVA: 0x00012B30 File Offset: 0x00010D30
		public IBookmarkMargin BookmarkMargin
		{
			get
			{
				return this.textEditor.TextArea.TextView.Services.GetService(typeof(IBookmarkMargin)) as IBookmarkMargin;
			}
		}

		// Token: 0x0600025A RID: 602 RVA: 0x00012B5C File Offset: 0x00010D5C
		public void Focus(AddPointModel model)
		{
			if (model.IsSelfDefinition)
			{
				FunctionSelectedModel functionSelectedModel = new FunctionSelectedModel
				{
					Name = model.FunctionName,
					Target = string.Format("{0} {1}(", model.Type.ToString(), model.FunctionNameWithoutParameter),
					ProgramKey = this.ProgramKey,
					Type = DefinitionType.FUNCTION
				};
				this._codeEditorManager.SearchBlock(functionSelectedModel);
				return;
			}
			SegmentObject segmentObject = this.textEditor.Document.SectionProvider.Find(model.ID);
			if (null == segmentObject)
			{
				return;
			}
			this.textEditor.ScrollTo(segmentObject.Offset);
		}

		// Token: 0x0600025B RID: 603 RVA: 0x00012C04 File Offset: 0x00010E04
		public void SectionFocus(Guid SectionID)
		{
			SegmentObject segmentObject = this.textEditor.Document.SectionProvider.Find(ContentType.SEC, SectionID);
			if (null == segmentObject)
			{
				return;
			}
			this.textEditor.ScrollTo(segmentObject.Offset);
		}

		// Token: 0x0600025C RID: 604 RVA: 0x00012C44 File Offset: 0x00010E44
		private void button_Click(object sender, RoutedEventArgs e)
		{
			this._codeEditorManager.OnRefreshScreen(this.ProgramKey);
			ContentType mode = this.textDiffEditor.Mode;
		}

		// Token: 0x0600025D RID: 605 RVA: 0x00012CC0 File Offset: 0x00010EC0
		private void custom_MouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			DataGrid dataGrid = sender as DataGrid;
			if (dataGrid == null)
			{
				return;
			}
			ADPModel adpmodel = dataGrid.SelectedItem as ADPModel;
			if (adpmodel == null)
			{
				return;
			}
			string name = adpmodel.Name;
			if (adpmodel.Type == "SEC")
			{
				SectionModel sectionModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Sections.Where<SectionModel>((SectionModel s) => s.Name == name && s.Status != Status.DELETE.Description()).FirstOrDefault<SectionModel>();
				if (sectionModel != null)
				{
					this.SectionFocus(sectionModel.ID);
					return;
				}
			}
			else
			{
				AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => ap.Name == name && (ap.Status & Status.DELETE) != Status.DELETE).FirstOrDefault<AddPointModel>();
				if (addPointModel != null)
				{
					this.Focus(addPointModel);
				}
			}
		}

		// Token: 0x0600025E RID: 606 RVA: 0x00012DF4 File Offset: 0x00010FF4
		private void merge_MouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			DataGrid dataGrid = sender as DataGrid;
			if (dataGrid == null)
			{
				return;
			}
			XElement xelement = dataGrid.SelectedItem as XElement;
			if (xelement == null)
			{
				return;
			}
			string name = xelement.Attribute("name").Value;
			string text;
			if ((text = xelement.Name.LocalName.ToLower()) != null)
			{
				if (!(text == "section"))
				{
					if (!(text == "point"))
					{
						return;
					}
					AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel ap) => ap.Name == name && (ap.Status & Status.DELETE) != Status.DELETE).FirstOrDefault<AddPointModel>();
					if (addPointModel != null)
					{
						this.Focus(addPointModel);
					}
				}
				else
				{
					SectionModel sectionModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Sections.Where<SectionModel>((SectionModel s) => s.Name == name && s.Status != Status.DELETE.Description()).FirstOrDefault<SectionModel>();
					if (sectionModel != null)
					{
						this.SectionFocus(sectionModel.ID);
						return;
					}
				}
			}
		}

		// Token: 0x0600025F RID: 607 RVA: 0x00012F00 File Offset: 0x00011100
		private void checkBox1_PreviewMouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			if (this.checkBox1.IsChecked != true)
			{
				this._codeEditorManager.isScrollSynced = true;
				return;
			}
			this._codeEditorManager.isScrollSynced = false;
		}

		// Token: 0x06000260 RID: 608 RVA: 0x00012F4C File Offset: 0x0001114C
		private void SwichToTopstd_PreviewMouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			CheckBox checkBox = sender as CheckBox;
			if (checkBox == null)
			{
				return;
			}
			this._codeEditorManager.SaveContent();
			this.textEditor.TextArea.TextView.Redraw();
			this.textEditor.Document.UndoStack.ClearAll();
			ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsTopstdMode = !(checkBox.IsChecked == true);
		}

		// Token: 0x06000261 RID: 609 RVA: 0x00012FD0 File Offset: 0x000111D0
		private void viewAdpDiffButton_Click(object sender, RoutedEventArgs e)
		{
			ADPModel adpmodel = this.customDataGrid.SelectedItem as ADPModel;
			if (adpmodel == null)
			{
				return;
			}
			StandardVersionComparisonWindow.This.Show(this.ProgramKey, adpmodel);
		}

		// Token: 0x14000003 RID: 3
		// (add) Token: 0x06000262 RID: 610 RVA: 0x00013004 File Offset: 0x00011204
		// (remove) Token: 0x06000263 RID: 611 RVA: 0x0001303C File Offset: 0x0001123C
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000264 RID: 612 RVA: 0x00013071 File Offset: 0x00011271
		public void OnPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
		}

		// Token: 0x06000265 RID: 613 RVA: 0x00013090 File Offset: 0x00011290
		public string GetSelectionText()
		{
			if (SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff)
			{
				if (this.textDiffEditor != null && this.textDiffEditor.IsKeyboardFocusWithin)
				{
					return this.textDiffEditor.TextArea.Selection.GetText();
				}
				if (!this.textEditor.IsKeyboardFocusWithin)
				{
					return string.Empty;
				}
				return this.textEditor.TextArea.Selection.GetText();
			}
			else
			{
				if (this.textEditor == null)
				{
					return string.Empty;
				}
				return this.textEditor.TextArea.Selection.GetText();
			}
		}

		// Token: 0x06000266 RID: 614 RVA: 0x0001312C File Offset: 0x0001132C
		private void OnToNextNormalization(object sender, ExecutedRoutedEventArgs e)
		{
			ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ClearNormalizationSearchResult();
			NormalizationSearchResult normalizationSearchResult = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).CurrentSearchResult;
			string infoXML = SettingManager.Get().GetTzpManger(this.ProgramKey).infoXML;
			SearchKeywordEventArgs e2 = new SearchKeywordEventArgs(this.ProgramKey, infoXML, DiffType.Normal);
			EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Publish(e2);
			SearchKeywordEventArgs e3 = new SearchKeywordEventArgs(this.ProgramKey, infoXML, DiffType.Diff);
			EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Publish(e3);
			List<NormalizationSearchResult> results = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Results;
			if (results.Count > 0)
			{
				CompareResult compareResult = new CompareResult();
				if (compareResult.Compare(normalizationSearchResult, results[results.Count - 1]).Equals(0))
				{
					normalizationSearchResult = results[0];
				}
				else
				{
					for (int i = 0; i < results.Count<NormalizationSearchResult>(); i++)
					{
						if (this.textDiffEditor.CaretOffset < results[i].StartOffset)
						{
							normalizationSearchResult = results[i];
							break;
						}
					}
				}
				ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).CurrentSearchResult = normalizationSearchResult;
				this.textDiffEditor.Select(normalizationSearchResult.StartOffset, normalizationSearchResult.Length);
				this.textDiffEditor.TextArea.Caret.Offset = normalizationSearchResult.StartOffset;
				this.textDiffEditor.TextArea.Caret.BringCaretToView();
			}
		}

		// Token: 0x06000267 RID: 615 RVA: 0x000132AC File Offset: 0x000114AC
		private void CanToNextNormalization(object sender, CanExecuteRoutedEventArgs e)
		{
			List<NormalizationSearchResult> results = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Results;
			e.CanExecute = SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff && results.Count > 0;
		}

		// Token: 0x06000268 RID: 616 RVA: 0x000132F8 File Offset: 0x000114F8
		private void OnPreviousNormalization(object sender, ExecutedRoutedEventArgs e)
		{
			ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ClearNormalizationSearchResult();
			NormalizationSearchResult normalizationSearchResult = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).CurrentSearchResult;
			string infoXML = SettingManager.Get().GetTzpManger(this.ProgramKey).infoXML;
			SearchKeywordEventArgs e2 = new SearchKeywordEventArgs(this.ProgramKey, infoXML, DiffType.Normal);
			EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Publish(e2);
			SearchKeywordEventArgs e3 = new SearchKeywordEventArgs(this.ProgramKey, infoXML, DiffType.Diff);
			EventAggregatorManager.Global.GetEvent<DiffNormalizationSearchEvent>().Publish(e3);
			List<NormalizationSearchResult> results = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Results;
			if (results.Count > 0)
			{
				CompareResult compareResult = new CompareResult();
				if (compareResult.Compare(normalizationSearchResult, results[0]).Equals(0))
				{
					normalizationSearchResult = results[results.Count<NormalizationSearchResult>() - 1];
				}
				else
				{
					for (int i = 0; i < results.Count<NormalizationSearchResult>(); i++)
					{
						if (this.textDiffEditor.CaretOffset > results[results.Count<NormalizationSearchResult>() - i - 1].StartOffset)
						{
							normalizationSearchResult = results[results.Count<NormalizationSearchResult>() - i - 1];
							break;
						}
					}
				}
				ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).CurrentSearchResult = normalizationSearchResult;
				this.textDiffEditor.Select(normalizationSearchResult.StartOffset, normalizationSearchResult.Length);
				this.textDiffEditor.TextArea.Caret.Offset = normalizationSearchResult.StartOffset;
				this.textDiffEditor.TextArea.Caret.BringCaretToView();
			}
		}

		// Token: 0x06000269 RID: 617 RVA: 0x0001348C File Offset: 0x0001168C
		private void CanToPreviousNormalization(object sender, CanExecuteRoutedEventArgs e)
		{
			List<NormalizationSearchResult> results = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Results;
			e.CanExecute = SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff && results.Count > 0;
		}

		// Token: 0x0600026D RID: 621 RVA: 0x000136D8 File Offset: 0x000118D8
		[DebuggerNonUserCode]
		[EditorBrowsable(EditorBrowsableState.Never)]
		[GeneratedCode("PresentationBuildTasks", "4.0.0.0")]
		void IStyleConnector.Connect(int connectionId, object target)
		{
			if (connectionId != 7)
			{
				return;
			}
			((Button)target).Click += this.viewAdpDiffButton_Click;
		}

		// Token: 0x040000FD RID: 253
		private CodeEditorManager _codeEditorManager;

		// Token: 0x040000FE RID: 254
		private StructureHelper _structureHelper;

		// Token: 0x040000FF RID: 255
		private ViEditMode _viManager;
	}
}
