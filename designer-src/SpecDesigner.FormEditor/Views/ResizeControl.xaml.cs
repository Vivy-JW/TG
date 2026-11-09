using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Markup;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x0200004E RID: 78
	public partial class ResizeControl : UserControl
	{
		// Token: 0x060002CF RID: 719 RVA: 0x0000E238 File Offset: 0x0000C438
		public ResizeControl()
		{
			this.InitializeComponent();
			this.BottomLeftRect.AddHandler(Thumb.DragDeltaEvent, new DragDeltaEventHandler(this.HandleBottomLeftRect));
			this.BottomRect.AddHandler(Thumb.DragDeltaEvent, new DragDeltaEventHandler(this.HandleBottomRect));
			this.BottomRightRect.AddHandler(Thumb.DragDeltaEvent, new DragDeltaEventHandler(this.HandleBottomRightRect));
			this.TopLeftRect.AddHandler(Thumb.DragDeltaEvent, new DragDeltaEventHandler(this.HandleTopLeftRect));
			this.TopRect.AddHandler(Thumb.DragDeltaEvent, new DragDeltaEventHandler(this.HandleTopRect));
			this.TopRightRect.AddHandler(Thumb.DragDeltaEvent, new DragDeltaEventHandler(this.HandleTopRightRect));
			this.LeftRect.AddHandler(Thumb.DragDeltaEvent, new DragDeltaEventHandler(this.HandleLeftRect));
			this.RightRect.AddHandler(Thumb.DragDeltaEvent, new DragDeltaEventHandler(this.HandleRightRect));
			this.BottomLeftRect.AddHandler(Thumb.DragCompletedEvent, new DragCompletedEventHandler(this.WhenDragCompleted));
			this.BottomRect.AddHandler(Thumb.DragCompletedEvent, new DragCompletedEventHandler(this.WhenDragCompleted));
			this.BottomRightRect.AddHandler(Thumb.DragCompletedEvent, new DragCompletedEventHandler(this.WhenDragCompleted));
			this.TopLeftRect.AddHandler(Thumb.DragCompletedEvent, new DragCompletedEventHandler(this.WhenDragCompleted));
			this.TopRect.AddHandler(Thumb.DragCompletedEvent, new DragCompletedEventHandler(this.WhenDragCompleted));
			this.TopRightRect.AddHandler(Thumb.DragCompletedEvent, new DragCompletedEventHandler(this.WhenDragCompleted));
			this.LeftRect.AddHandler(Thumb.DragCompletedEvent, new DragCompletedEventHandler(this.WhenDragCompleted));
			this.RightRect.AddHandler(Thumb.DragCompletedEvent, new DragCompletedEventHandler(this.WhenDragCompleted));
			this.BottomLeftRect.AddHandler(Thumb.DragStartedEvent, new DragStartedEventHandler(this.WhenDragStarted));
			this.BottomRect.AddHandler(Thumb.DragStartedEvent, new DragStartedEventHandler(this.WhenDragStarted));
			this.BottomRightRect.AddHandler(Thumb.DragStartedEvent, new DragStartedEventHandler(this.WhenDragStarted));
			this.TopLeftRect.AddHandler(Thumb.DragStartedEvent, new DragStartedEventHandler(this.WhenDragStarted));
			this.TopRect.AddHandler(Thumb.DragStartedEvent, new DragStartedEventHandler(this.WhenDragStarted));
			this.TopRightRect.AddHandler(Thumb.DragStartedEvent, new DragStartedEventHandler(this.WhenDragStarted));
			this.LeftRect.AddHandler(Thumb.DragStartedEvent, new DragStartedEventHandler(this.WhenDragStarted));
			this.RightRect.AddHandler(Thumb.DragStartedEvent, new DragStartedEventHandler(this.WhenDragStarted));
		}

		// Token: 0x060002D0 RID: 720 RVA: 0x0000E4F4 File Offset: 0x0000C6F4
		private void WhenDragStarted(object sender, DragStartedEventArgs e)
		{
			if (this._model == null)
			{
				this._model = base.DataContext as XmlElement;
			}
			ComponentHelper.Get(this._model.Key).IsSizeChanging = true;
			this._changeSizeUndoRedoCommand = new ChangeSizeUndoRedoCommand(ComponentHelper.Get(this._model.Key).SelectedObjects);
			this._sizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this._changeSizeUndoRedoCommand, false);
			SettingManager.Get().GetUndoRedoManager(this._model.Key).StartGroup(this._sizeComplexUndoRedoCommand);
		}

		// Token: 0x060002D1 RID: 721 RVA: 0x0000E584 File Offset: 0x0000C784
		private void WhenDragCompleted(object sender, DragCompletedEventArgs e)
		{
			if (this._model == null || this._sizeComplexUndoRedoCommand == null)
			{
				return;
			}
			foreach (XmlElement xmlElement in ComponentHelper.Get(this._model.Key).SelectedObjects)
			{
				this._changeSizeUndoRedoCommand.SetFinalSize(xmlElement, xmlElement.GridX, xmlElement.GridY, xmlElement.GridWidth, xmlElement.GridHeight);
			}
			SettingManager.Get().GetUndoRedoManager(this._model.Key).EndGroup(this._sizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._model.Key).IsSizeChanging = false;
		}

		// Token: 0x060002D2 RID: 722 RVA: 0x0000E644 File Offset: 0x0000C844
		private void HandleBottomRect(object sender, DragDeltaEventArgs e)
		{
			e.Handled = true;
			if (this._model == null)
			{
				return;
			}
			this.SetHeight(e.VerticalChange);
		}

		// Token: 0x060002D3 RID: 723 RVA: 0x0000E662 File Offset: 0x0000C862
		private void HandleBottomRightRect(object sender, DragDeltaEventArgs e)
		{
			e.Handled = true;
			if (this._model == null)
			{
				return;
			}
			this.SetWidth(e.HorizontalChange);
			this.SetHeight(e.VerticalChange);
		}

		// Token: 0x060002D4 RID: 724 RVA: 0x0000E68C File Offset: 0x0000C88C
		private void HandleRightRect(object sender, DragDeltaEventArgs e)
		{
			e.Handled = true;
			if (this._model == null)
			{
				return;
			}
			this.SetWidth(e.HorizontalChange);
		}

		// Token: 0x060002D5 RID: 725 RVA: 0x0000E6AA File Offset: 0x0000C8AA
		private void HandleLeftRect(object sender, DragDeltaEventArgs e)
		{
			e.Handled = true;
			if (this._model == null)
			{
				return;
			}
			this.SetX(e.HorizontalChange);
		}

		// Token: 0x060002D6 RID: 726 RVA: 0x0000E6C8 File Offset: 0x0000C8C8
		private void HandleBottomLeftRect(object sender, DragDeltaEventArgs e)
		{
			e.Handled = true;
			if (this._model == null)
			{
				return;
			}
			this.SetX(e.HorizontalChange);
			this.SetHeight(e.VerticalChange);
		}

		// Token: 0x060002D7 RID: 727 RVA: 0x0000E6F2 File Offset: 0x0000C8F2
		private void HandleTopRect(object sender, DragDeltaEventArgs e)
		{
			e.Handled = true;
			if (this._model == null)
			{
				return;
			}
			this.SetY(e.VerticalChange);
		}

		// Token: 0x060002D8 RID: 728 RVA: 0x0000E710 File Offset: 0x0000C910
		private void HandleTopLeftRect(object sender, DragDeltaEventArgs e)
		{
			e.Handled = true;
			if (this._model == null)
			{
				return;
			}
			this.SetX(e.HorizontalChange);
			this.SetY(e.VerticalChange);
		}

		// Token: 0x060002D9 RID: 729 RVA: 0x0000E73A File Offset: 0x0000C93A
		private void HandleTopRightRect(object sender, DragDeltaEventArgs e)
		{
			e.Handled = true;
			if (this._model == null)
			{
				return;
			}
			this.SetWidth(e.HorizontalChange);
			this.SetY(e.VerticalChange);
		}

		// Token: 0x060002DA RID: 730 RVA: 0x0000E764 File Offset: 0x0000C964
		private void SetX(double dragDeltaChange)
		{
			dragDeltaChange = Math.Min((double)(this._model.Width - this._model.MinGridWidth * FormDesignSetting.UnitWidth), dragDeltaChange);
			int num = FormDesignSetting.TransformToGridWidth(dragDeltaChange);
			if (num == 0)
			{
				return;
			}
			foreach (XmlElement xmlElement in ComponentHelper.Get(this._model.Key).SelectedObjects)
			{
				if (num > 0)
				{
					xmlElement.GridWidth -= num;
					xmlElement.GridX += num;
				}
				else if (num < 0)
				{
					if (xmlElement.GridX + num < xmlElement.MinGridX)
					{
						break;
					}
					xmlElement.GridX += num;
					xmlElement.GridWidth -= num;
				}
			}
		}

		// Token: 0x060002DB RID: 731 RVA: 0x0000E840 File Offset: 0x0000CA40
		private void SetY(double dragDeltaChange)
		{
			dragDeltaChange = Math.Min((double)(this._model.Height - this._model.MinGridHeight * FormDesignSetting.UnitHeight), dragDeltaChange);
			int num = FormDesignSetting.TransformToGridHeight(dragDeltaChange);
			if (num == 0)
			{
				return;
			}
			foreach (XmlElement xmlElement in ComponentHelper.Get(this._model.Key).SelectedObjects)
			{
				if (num > 0)
				{
					xmlElement.GridHeight -= num;
					xmlElement.GridY += num;
				}
				else if (num < 0)
				{
					if (xmlElement.GridY + num < xmlElement.MinGridY)
					{
						break;
					}
					xmlElement.GridY += num;
					xmlElement.GridHeight -= num;
				}
			}
		}

		// Token: 0x060002DC RID: 732 RVA: 0x0000E91C File Offset: 0x0000CB1C
		private void SetWidth(double dragDeltaChange)
		{
			int num = FormDesignSetting.TransformToGridWidth(dragDeltaChange);
			if (num == 0)
			{
				return;
			}
			foreach (XmlElement xmlElement in ComponentHelper.Get(this._model.Key).SelectedObjects)
			{
				xmlElement.GridWidth += num;
			}
		}

		// Token: 0x060002DD RID: 733 RVA: 0x0000E98C File Offset: 0x0000CB8C
		private void SetHeight(double dragDeltaChange)
		{
			int num = FormDesignSetting.TransformToGridHeight(dragDeltaChange);
			if (num == 0)
			{
				return;
			}
			foreach (XmlElement xmlElement in ComponentHelper.Get(this._model.Key).SelectedObjects)
			{
				xmlElement.GridHeight += num;
			}
		}

		// Token: 0x0400018A RID: 394
		private XmlElement _model;

		// Token: 0x0400018B RID: 395
		private FormSizeComplexUndoRedoCommand _sizeComplexUndoRedoCommand;

		// Token: 0x0400018C RID: 396
		private ChangeSizeUndoRedoCommand _changeSizeUndoRedoCommand;
	}
}
