using System;
using System.Windows;
using System.Windows.Media.Imaging;
using SpecDesigner.CodeEditWindow.View;
using SpecDesignerCommon;
using SpecDesignerCommon.Bookmark;
using SpecDesignerCommon.Events;

namespace SpecDesigner.ViewModels
{
	// Token: 0x02000034 RID: 52
	public class CodeViewModel : FileViewModel
	{
		// Token: 0x0600029F RID: 671 RVA: 0x0000C26C File Offset: 0x0000A46C
		public CodeViewModel(PackageKey key)
			: base(key)
		{
			base.UI = new CodeEditorMainWindow(key);
			base.IconSource = new BitmapImage(new Uri("/Images/document_4gl.png", UriKind.Relative));
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingCodeEvent>().Publish(base.Key);
			EventAggregatorManager.Get(base.Key).GetEvent<CodeChangedEvent>().Subscribe(new Action<PackageKey>(this.OnCodeChanged));
			EventAggregatorManager.Global.GetEvent<SearchCodeFromFormEvent>().Subscribe(new Action<FieldArgs>(this.SearchCodeFromForm));
			EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Subscribe(new Action<SearchResultInfo>(this.OnSearchResultSelected));
		}

		// Token: 0x060002A0 RID: 672 RVA: 0x0000C314 File Offset: 0x0000A514
		public override void OnClose()
		{
			if (EditorWorkspace.This.Close(this))
			{
				EventAggregatorManager.Get(base.Key).GetEvent<CodeChangedEvent>().Unsubscribe(new Action<PackageKey>(this.OnCodeChanged));
				EventAggregatorManager.Global.GetEvent<SearchCodeFromFormEvent>().Unsubscribe(new Action<FieldArgs>(this.SearchCodeFromForm));
				EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Unsubscribe(new Action<SearchResultInfo>(this.OnSearchResultSelected));
				base.OnClose();
			}
			GC.Collect();
		}

		// Token: 0x060002A1 RID: 673 RVA: 0x0000C390 File Offset: 0x0000A590
		private void SearchCodeFromForm(FieldArgs args)
		{
			if (args.ProgramKey.Program != base.Key.Program)
			{
				return;
			}
			PackageKey packageKey = new PackageKey(args.ProgramKey.Program, TzpType.Code);
			FieldArgs fieldArgs = new FieldArgs(args.Field, packageKey);
			EditorWorkspace.This.SetActiveDocumentFromKey(packageKey);
			EventAggregatorManager.Global.GetEvent<SearchCodeEvent>().Publish(fieldArgs);
		}

		// Token: 0x060002A2 RID: 674 RVA: 0x0000C3F7 File Offset: 0x0000A5F7
		private void OnSearchResultSelected(SearchResultInfo info)
		{
			if (info.ProgramKey == base.Key)
			{
				base.IsActive = true;
			}
		}

		// Token: 0x060002A3 RID: 675 RVA: 0x0000C413 File Offset: 0x0000A613
		private void OnCodeChanged(PackageKey key)
		{
			if (base.Key == key)
			{
				base.IsModified = true;
			}
		}

		// Token: 0x1700009F RID: 159
		// (get) Token: 0x060002A4 RID: 676 RVA: 0x0000C42C File Offset: 0x0000A62C
		public override FrameworkElement PropertiesUI
		{
			get
			{
				FileViewModel fileViewModel = EditorWorkspace.This.FindFileViewModel(new PackageKey(base.Key.Program, TzpType.Form));
				if (fileViewModel != null)
				{
					return fileViewModel.PropertiesUI;
				}
				return null;
			}
		}

		// Token: 0x170000A0 RID: 160
		// (get) Token: 0x060002A5 RID: 677 RVA: 0x0000C460 File Offset: 0x0000A660
		public override FrameworkElement SpecUI
		{
			get
			{
				FileViewModel fileViewModel = EditorWorkspace.This.FindFileViewModel(new PackageKey(base.Key.Program, TzpType.Form));
				if (fileViewModel != null)
				{
					return fileViewModel.SpecUI;
				}
				return null;
			}
		}

		// Token: 0x170000A1 RID: 161
		// (get) Token: 0x060002A6 RID: 678 RVA: 0x0000C494 File Offset: 0x0000A694
		public override FrameworkElement Structure
		{
			get
			{
				FileViewModel fileViewModel = EditorWorkspace.This.FindFileViewModel(new PackageKey(base.Key.Program, TzpType.Form));
				if (fileViewModel != null)
				{
					return fileViewModel.Structure;
				}
				return null;
			}
		}

		// Token: 0x170000A2 RID: 162
		// (get) Token: 0x060002A7 RID: 679 RVA: 0x0000C4C8 File Offset: 0x0000A6C8
		public override IBookmarkMargin BookmarkManager
		{
			get
			{
				CodeEditorMainWindow codeEditorMainWindow = base.UI as CodeEditorMainWindow;
				if (codeEditorMainWindow != null)
				{
					return codeEditorMainWindow.BookmarkMargin;
				}
				return null;
			}
		}

		// Token: 0x170000A3 RID: 163
		// (get) Token: 0x060002A8 RID: 680 RVA: 0x0000C4EC File Offset: 0x0000A6EC
		public override string SelectionContent
		{
			get
			{
				CodeEditorMainWindow codeEditorMainWindow = base.UI as CodeEditorMainWindow;
				if (codeEditorMainWindow == null)
				{
					return string.Empty;
				}
				return codeEditorMainWindow.GetSelectionText();
			}
		}
	}
}
