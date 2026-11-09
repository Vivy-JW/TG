using System;
using System.Windows;
using System.Windows.Media.Imaging;
using SpecDesigner.FormEditor;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.ViewModels
{
	// Token: 0x02000033 RID: 51
	public class FormViewModel : FileViewModel
	{
		// Token: 0x06000296 RID: 662 RVA: 0x0000C050 File Offset: 0x0000A250
		public FormViewModel(PackageKey key)
			: base(key)
		{
			this._formEditor = new FormEditorMainWindow(key);
			base.UI = this._formEditor;
			base.IconSource = new BitmapImage(new Uri("/Images/document_4fd.png", UriKind.Relative));
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingFormEvent>().Publish(base.Key);
			EventAggregatorManager.Get(base.Key).GetEvent<SpecPropertiesChangedEvent>().Subscribe(new Action<PackageKey>(this.OnSpecPropertiesChanged));
			EventAggregatorManager.Get(base.Key).GetEvent<FormPropertyChangedEvent>().Subscribe(new Action<PackageKey>(this.OnFormPropertyChanged));
		}

		// Token: 0x06000297 RID: 663 RVA: 0x0000C0EC File Offset: 0x0000A2EC
		public override void OnClose()
		{
			if (EditorWorkspace.This.Close(this))
			{
				SettingManager.Get().undoRedoManagerMap.Remove(base.Key);
				EventAggregatorManager.Get(base.Key).GetEvent<SpecPropertiesChangedEvent>().Unsubscribe(new Action<PackageKey>(this.OnSpecPropertiesChanged));
				EventAggregatorManager.Get(base.Key).GetEvent<FormPropertyChangedEvent>().Unsubscribe(new Action<PackageKey>(this.OnFormPropertyChanged));
				EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Unsubscribe(new Action<SearchResultInfo>(this.OnSearchResultSelected));
				base.OnClose();
			}
			GC.Collect();
		}

		// Token: 0x06000298 RID: 664 RVA: 0x0000C184 File Offset: 0x0000A384
		private void OnSpecPropertiesChanged(PackageKey key)
		{
			if (key.Equals(base.Key))
			{
				base.IsModified = true;
			}
		}

		// Token: 0x06000299 RID: 665 RVA: 0x0000C19B File Offset: 0x0000A39B
		private void OnFormPropertyChanged(PackageKey key)
		{
			if (key.Equals(base.Key))
			{
				base.IsModified = true;
			}
		}

		// Token: 0x0600029A RID: 666 RVA: 0x0000C1B2 File Offset: 0x0000A3B2
		private void OnSearchResultSelected(SearchResultInfo info)
		{
			if (base.Key.Equals(info.ProgramKey))
			{
				base.IsActive = true;
			}
		}

		// Token: 0x1700009B RID: 155
		// (get) Token: 0x0600029B RID: 667 RVA: 0x0000C1CE File Offset: 0x0000A3CE
		public override FrameworkElement PropertiesUI
		{
			get
			{
				if (SettingManager.Get().GetTzpManger(base.Key).IsSimpleForm)
				{
					return null;
				}
				if (this._formEditor != null)
				{
					return this._formEditor.FormPropertiesContent;
				}
				return null;
			}
		}

		// Token: 0x1700009C RID: 156
		// (get) Token: 0x0600029C RID: 668 RVA: 0x0000C1FE File Offset: 0x0000A3FE
		public override FrameworkElement Structure
		{
			get
			{
				if (SettingManager.Get().GetTzpManger(base.Key).IsSimpleForm)
				{
					return null;
				}
				if (this._formEditor != null)
				{
					return this._formEditor.FormStructureContent;
				}
				return null;
			}
		}

		// Token: 0x1700009D RID: 157
		// (get) Token: 0x0600029D RID: 669 RVA: 0x0000C22E File Offset: 0x0000A42E
		public override FrameworkElement SpecUI
		{
			get
			{
				if (SettingManager.Get().GetTzpManger(base.Key).IsSimpleForm)
				{
					return null;
				}
				if (this._formEditor != null)
				{
					return this._formEditor.FormSpecEditorContent;
				}
				return null;
			}
		}

		// Token: 0x1700009E RID: 158
		// (get) Token: 0x0600029E RID: 670 RVA: 0x0000C25E File Offset: 0x0000A45E
		public override FrameworkElement DatabaseSource
		{
			get
			{
				return this._formEditor.FormDatabaseSource;
			}
		}

		// Token: 0x0400017F RID: 383
		private FormEditorMainWindow _formEditor;
	}
}
